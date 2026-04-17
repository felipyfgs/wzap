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
end
