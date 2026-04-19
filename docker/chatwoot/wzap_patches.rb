# =============================================================================
# wzap_patches.rb — Monkey patches no Chatwoot para integração com wzap
# =============================================================================
# Este arquivo é montado como `/app/config/initializers/wzap_patches.rb` via
# docker-compose e carregado automaticamente no boot do Rails.
#
# Motivação:
# O wzap emula a WhatsApp Cloud API da Meta mas usa `whatsmeow` sob o capô
# (protocolo WhatsApp Web), que NÃO tem a restrição de "janela de 24h" da
# Cloud API oficial. O Chatwoot por padrão bloqueia respostas livres fora
# dessa janela, forçando o uso de templates aprovados — comportamento
# inadequado quando a inbox é wzap.
#
# Controle via env:
#   WZAP_BYPASS_24H_WINDOW=true   → desativa janela de 24h para Channel::Whatsapp
#                                   com provider `whatsapp_cloud`
#   (ausente ou false)            → comportamento padrão do Chatwoot
# =============================================================================

Rails.application.config.after_initialize do
  if ENV['WZAP_BYPASS_24H_WINDOW'].to_s.downcase == 'true'
    Rails.logger.info '[wzap] bypass da janela de 24h habilitado para Channel::Whatsapp (whatsapp_cloud)'

    Conversations::MessageWindowService.class_eval do
      alias_method :__wzap_original_can_reply?, :can_reply?

      def can_reply?
        # Inboxes WhatsApp Cloud atendidas pelo wzap não têm janela de 24h
        # (protocolo WhatsApp Web via whatsmeow). Retorna sempre true.
        channel = @conversation.inbox.channel
        if @conversation.inbox.channel_type == 'Channel::Whatsapp' &&
           channel.respond_to?(:provider) && channel.provider == 'whatsapp_cloud'
          return true
        end

        __wzap_original_can_reply?
      end
    end
  end

  # =============================================================================
  # Propagação de deleção de mensagem para o WhatsApp via wzap (Cloud inbox)
  # =============================================================================
  # Chatwoot vanilla só faz soft-delete local (content_attributes.deleted=true)
  # e o provider whatsapp_cloud_service.rb não implementa delete_message — então
  # o cliente final continua vendo a mensagem no WhatsApp depois que o agente
  # a apaga no Chatwoot. Este hook fecha a lacuna.
  #
  # Estratégia: after_update em Message detecta a transição para deleted=true,
  # e POSTa para o endpoint Cloud do wzap usando o mesmo shape que o Chatwoot
  # já usa para "mark as read": { messaging_product, status, message_id }. Do
  # outro lado o wzap (CloudAPIHandler.handleDeleteMessage) chama BuildRevoke
  # via whatsmeow.
  #
  # Apenas para Channel::Whatsapp provider=whatsapp_cloud. DM direto ao endpoint
  # do próprio canal (api_base_path + phone_number_id) — o token já é o que o
  # Chatwoot usa normalmente pra todo o resto do provider.
  Message.class_eval do
    after_update :__wzap_propagate_delete_to_wa

    private

    def __wzap_propagate_delete_to_wa
      return unless saved_change_to_content_attributes?
      return unless content_attributes.is_a?(Hash) && content_attributes['deleted'] == true

      ibox = inbox
      return unless ibox && ibox.channel_type == 'Channel::Whatsapp'

      channel = ibox.channel
      return unless channel.respond_to?(:provider) && channel.provider == 'whatsapp_cloud'

      waid = source_id.to_s.sub(/^WAID:/, '')
      return if waid.blank?

      provider_cfg = channel.provider_config || {}
      phone_id = provider_cfg['phone_number_id']
      api_key = provider_cfg['api_key']
      return if phone_id.blank?

      api_base = ENV.fetch('WHATSAPP_CLOUD_BASE_URL', 'https://graph.facebook.com')
      url = "#{api_base}/v14.0/#{phone_id}/messages"
      body = {
        messaging_product: 'whatsapp',
        status: 'deleted',
        message_id: waid
      }
      headers = {
        'Content-Type' => 'application/json',
        'Authorization' => "Bearer #{api_key}"
      }

      Thread.new do
        begin
          HTTParty.post(url, body: body.to_json, headers: headers, timeout: 10)
        rescue StandardError => e
          Rails.logger.warn "[wzap] falha ao propagar delete wa msg=#{waid}: #{e.message}"
        end
      end
    rescue StandardError => e
      Rails.logger.warn "[wzap] erro em __wzap_propagate_delete_to_wa: #{e.message}"
    end
  end
  Rails.logger.info '[wzap] monkey patch carregado: Message#__wzap_propagate_delete_to_wa'
end
