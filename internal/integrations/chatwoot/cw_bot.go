package chatwoot

import (
	"context"
	"strings"

	"wzap/internal/logger"
)

func (s *Service) processBotCommand(ctx context.Context, cfg *Config, content string) error {
	command := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(content), "/"))
	command = strings.ToLower(command)

	parts := strings.SplitN(command, ":", 2)
	cmd := parts[0]

	convID, err := s.ensureBotConv(ctx, cfg)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Msg("failed to get bot conversation for command")
		return err
	}

	client := s.clientFn(cfg)

	switch cmd {
	case "init", "iniciar", "connect", "conectar":
		if s.connector == nil {
			s.sendBotReply(ctx, client, convID, "⚠️ Conector de sessão não disponível.")
			return nil
		}
		if s.connector.IsConnected(cfg.SessionID) {
			s.sendBotReply(ctx, client, convID, "✅ Sessão já está conectada ao WhatsApp.")
			return nil
		}
		s.sendBotReply(ctx, client, convID, "⏳ Conectando ao WhatsApp...")
		if err := s.connector.Connect(ctx, cfg.SessionID); err != nil {
			s.sendBotReply(ctx, client, convID, "❌ Falha ao conectar: "+err.Error())
			return nil
		}

	case "status":
		if s.connector == nil {
			s.sendBotReply(ctx, client, convID, "⚠️ Conector de sessão não disponível.")
			return nil
		}
		if s.connector.IsConnected(cfg.SessionID) {
			s.sendBotReply(ctx, client, convID, "✅ Sessão conectada ao WhatsApp.")
		} else {
			s.sendBotReply(ctx, client, convID, "❌ Sessão desconectada.\n\nEnvie *init* para conectar.")
		}

	case "disconnect", "desconectar":
		if s.connector == nil {
			s.sendBotReply(ctx, client, convID, "⚠️ Conector de sessão não disponível.")
			return nil
		}
		s.sendBotReply(ctx, client, convID, "⏳ Desconectando do WhatsApp...")
		if err := s.connector.Disconnect(ctx, cfg.SessionID); err != nil {
			s.sendBotReply(ctx, client, convID, "❌ Falha ao desconectar: "+err.Error())
			return nil
		}
		s.sendBotReply(ctx, client, convID, "✅ Sessão desconectada com sucesso.")

	case "logout", "sair":
		if s.connector == nil {
			s.sendBotReply(ctx, client, convID, "⚠️ Conector de sessão não disponível.")
			return nil
		}
		s.sendBotReply(ctx, client, convID, "⏳ Desvinculando dispositivo...")
		if err := s.connector.Logout(ctx, cfg.SessionID); err != nil {
			s.sendBotReply(ctx, client, convID, "❌ Falha ao desvincular: "+err.Error())
			return nil
		}
		s.sendBotReply(ctx, client, convID, "✅ Dispositivo desvinculado. Envie *init* para reconectar com novo QR code.")

	case "clearcache", "limparcache":
		s.cache.DeleteConv(ctx, cfg.SessionID, "")
		s.missingConfig.Delete(cfg.SessionID)
		s.sendBotReply(ctx, client, convID, "🗑️ Cache limpo com sucesso.")

	case "help", "ajuda":
		s.sendBotReply(ctx, client, convID, botHelpMessage())

	default:
		s.sendBotReply(ctx, client, convID, "❓ Comando não reconhecido: *"+cmd+"*\n\n"+botHelpMessage())
	}

	return nil
}

func (s *Service) sendBotReply(ctx context.Context, client Client, convID int, content string) {
	_, _ = client.CreateMessage(ctx, convID, MessageReq{
		Content:     content,
		MessageType: "incoming",
	})
}

func botHelpMessage() string {
	return `📋 *Comandos disponíveis:*

• *init* ou *conectar* — Conectar ao WhatsApp
• *status* — Verificar status da conexão
• *disconnect* ou *desconectar* — Desconectar sessão
• *logout* ou *sair* — Desvincular dispositivo
• *clearcache* — Limpar cache
• *help* ou *ajuda* — Exibir esta mensagem`
}
