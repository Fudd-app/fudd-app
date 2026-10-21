package auth

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"time"
)

// smtpTimeout acota cada intento de envío para no bloquear el request HTTP.
const smtpTimeout = 20 * time.Second

type EmailSender interface {
	EnviarMagicLink(ctx context.Context, email, link string) error
}

type invitationEmailSender interface {
	EnviarInvitacion(ctx context.Context, email, link string) error
}

// EnviarInvitacion usa una plantilla específica cuando el proveedor la
// soporta. El fallback mantiene compatibilidad con emisores de pruebas.
func EnviarInvitacion(ctx context.Context, sender EmailSender, email, link string) error {
	if especializado, ok := sender.(invitationEmailSender); ok {
		return especializado.EnviarInvitacion(ctx, email, link)
	}
	return sender.EnviarMagicLink(ctx, email, link)
}

// LogEmailSender solo loguea el link para desarrollo local.
type LogEmailSender struct{}

func (s *LogEmailSender) EnviarMagicLink(ctx context.Context, email, link string) error {
	slog.InfoContext(ctx, "Enviando magic link por LOG (Desarrollo)", "email", email, "link", link)
	return nil
}

func (s *LogEmailSender) EnviarInvitacion(ctx context.Context, email, link string) error {
	slog.InfoContext(ctx, "Enviando invitación por LOG (Desarrollo)", "email", email, "link", link)
	return nil
}

// BrevoEmailSender envía correos transaccionales mediante la API HTTP de Brevo.
type BrevoEmailSender struct {
	apiKey      string
	senderName  string
	senderEmail string
}

func NuevoBrevoEmailSender(apiKey, senderName, senderEmail string) *BrevoEmailSender {
	return &BrevoEmailSender{
		apiKey:      apiKey,
		senderName:  senderName,
		senderEmail: senderEmail,
	}
}

func (s *BrevoEmailSender) EnviarMagicLink(ctx context.Context, toEmail, link string) error {
	return s.enviar(ctx, toEmail, "Tu enlace de acceso a fudd",
		fmt.Sprintf("<p>Hola,</p><p>Hacé clic en el siguiente enlace para ingresar a tu cuenta de fudd. Expira en 15 minutos:</p><p><a href=\"%s\">Ingresar a fudd</a></p><p>Si no solicitaste este enlace, podés ignorar este correo.</p>", link))
}

func (s *BrevoEmailSender) EnviarInvitacion(ctx context.Context, toEmail, link string) error {
	return s.enviar(ctx, toEmail, "Te invitaron al equipo de fudd",
		fmt.Sprintf("<p>Hola,</p><p>Te invitaron a formar parte de un equipo en fudd.</p><p><a href=\"%s\">Aceptar invitación con Google</a></p><p>El enlace vence en 48 horas y puede usarse una sola vez. Debés continuar con la misma dirección de email que recibió esta invitación.</p>", link))
}

func (s *BrevoEmailSender) enviar(ctx context.Context, toEmail, subject, htmlContent string) error {
	body := map[string]any{
		"sender": map[string]string{
			"name":  s.senderName,
			"email": s.senderEmail,
		},
		"to":          []map[string]string{{"email": toEmail}},
		"subject":     subject,
		"htmlContent": htmlContent,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("error serializando request de email: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.brevo.com/v3/smtp/email", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("error creando request a Brevo: %w", err)
	}
	req.Header.Set("api-key", s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error enviando request a Brevo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("error de Brevo: status %d", resp.StatusCode)
	}

	slog.InfoContext(ctx, "magic link enviado con éxito por Brevo", "email", toEmail)
	return nil
}

// SMTPEmailSender envía correos a través de cualquier servidor SMTP estándar (Gmail, Outlook, Brevo, SendGrid, etc.).
type SMTPEmailSender struct {
	host     string
	port     string
	user     string
	password string
	from     string
}

func NuevoSMTPEmailSender(host, port, user, password, from string) *SMTPEmailSender {
	if port == "" {
		port = "587"
	}
	if from == "" {
		from = user
	}
	return &SMTPEmailSender{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		from:     from,
	}
}

func (s *SMTPEmailSender) EnviarMagicLink(ctx context.Context, toEmail, link string) error {
	addr := net.JoinHostPort(s.host, s.port)
	subject := "Tu enlace de acceso a fudd"

	// SMTP_FROM puede venir como "Nombre <a@b.com>" o como "a@b.com".
	// La cabecera From: acepta las dos formas; el comando MAIL FROM solo la pelada.
	remitente := s.from
	if dir, err := mail.ParseAddress(s.from); err == nil {
		remitente = dir.Address
	}

	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html lang="es">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <style>
    body { font-family: 'Outfit', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #f5f3ec; margin: 0; padding: 24px; color: #12382c; }
    .card { max-width: 500px; margin: 0 auto; background: #ffffff; border-radius: 12px; border: 1px solid #e5e7eb; padding: 32px; box-shadow: 0 2px 8px rgba(0,0,0,0.04); }
    .logo { font-size: 24px; font-weight: 800; letter-spacing: -0.03em; color: #12382c; margin-bottom: 20px; }
    .btn { display: inline-block; background-color: #12382c; color: #f5f3ec; font-weight: 700; text-decoration: none; padding: 12px 24px; border-radius: 8px; margin: 20px 0; font-size: 15px; }
    .footer { font-size: 12px; color: #607166; margin-top: 24px; line-height: 1.5; }
    .link-alt { word-break: break-all; color: #ff6a3d; font-size: 12px; }
  </style>
</head>
<body>
  <div class="card">
    <div class="logo">fudd</div>
    <h2 style="font-size: 18px; margin: 0 0 12px 0;">Acceder a fudd</h2>
    <p style="font-size: 14px; line-height: 1.5; color: #283a2e;">
      Hacé clic en el siguiente botón para continuar. Este enlace es personal y de un solo uso:
    </p>
    <div style="text-align: center;">
      <a href="%s" class="btn">Continuar en fudd</a>
    </div>
    <p class="footer">
      O copiá y pegá este enlace en tu navegador:<br>
      <a href="%s" class="link-alt">%s</a>
    </p>
    <hr style="border: none; border-top: 1px solid #f3f4f6; margin: 24px 0;">
    <p class="footer" style="margin: 0;">
      Si no solicitaste este acceso, podés ignorar este correo de forma segura.
    </p>
  </div>
</body>
</html>`, link, link, link)

	msg := []byte(fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n\r\n"+
		"%s", s.from, toEmail, subject, htmlContent))

	var auth smtp.Auth
	if s.user != "" && s.password != "" {
		auth = smtp.PlainAuth("", s.user, s.password, s.host)
	}

	client, err := s.conectar(addr)
	if err != nil {
		return err
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("error autenticando SMTP como %s: %w", s.user, err)
		}
	}

	// El envelope sender debe ser la dirección pelada: "Nombre <a@b.com>" solo
	// vale en la cabecera From:. Zoho lo tolera, otros servidores lo rechazan.
	if err := client.Mail(remitente); err != nil {
		return fmt.Errorf("servidor SMTP rechazó el remitente %s: %w", remitente, err)
	}
	if err := client.Rcpt(toEmail); err != nil {
		return fmt.Errorf("servidor SMTP rechazó el destinatario %s: %w", toEmail, err)
	}

	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("error entregando el mensaje: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("error cerrando la sesión SMTP: %w", err)
	}

	slog.InfoContext(ctx, "magic link enviado con éxito vía SMTP", "email", toEmail, "smtp_host", s.host)
	return nil
}

// SMTP conserva el mismo transporte y plantilla HTML; el texto del enlace
// sigue siendo válido para aceptar la invitación y el backend impone las 48 h.
func (s *SMTPEmailSender) EnviarInvitacion(ctx context.Context, toEmail, link string) error {
	return s.EnviarMagicLink(ctx, toEmail, link)
}

// conectar abre la sesión SMTP con timeout explícito. Sin esto, un puerto
// filtrado por el firewall o el proveedor deja el request colgado hasta el
// timeout del sistema operativo (~21s en Windows) en lugar de fallar rápido.
func (s *SMTPEmailSender) conectar(addr string) (*smtp.Client, error) {
	conn, err := net.DialTimeout("tcp", addr, smtpTimeout)
	if err != nil {
		return nil, fmt.Errorf("no se pudo conectar al servidor SMTP %s: %w "+
			"(si es un timeout, probá SMTP_PORT=587 o revisá si el proveedor bloquea el puerto)", addr, err)
	}
	// Cubre handshake, AUTH y entrega del mensaje.
	_ = conn.SetDeadline(time.Now().Add(smtpTimeout))

	tlsConfig := &tls.Config{ServerName: s.host}

	if s.port == "465" {
		// Puerto 465: TLS implícito, la sesión arranca cifrada.
		tlsConn := tls.Client(conn, tlsConfig)
		if err := tlsConn.Handshake(); err != nil {
			conn.Close()
			return nil, fmt.Errorf("error en el handshake TLS con %s: %w", addr, err)
		}
		client, err := smtp.NewClient(tlsConn, s.host)
		if err != nil {
			tlsConn.Close()
			return nil, fmt.Errorf("error creando cliente SMTP: %w", err)
		}
		return client, nil
	}

	// Puertos 587 / 25: sesión en claro y se promueve con STARTTLS.
	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("error creando cliente SMTP: %w", err)
	}
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(tlsConfig); err != nil {
			client.Close()
			return nil, fmt.Errorf("error en STARTTLS con %s: %w", addr, err)
		}
	}
	return client, nil
}

// ResendEmailSender envía correos a través de la API REST de Resend.
type ResendEmailSender struct {
	apiKey    string
	fromEmail string
}

func NuevoResendEmailSender(apiKey, fromEmail string) *ResendEmailSender {
	return &ResendEmailSender{apiKey: apiKey, fromEmail: fromEmail}
}

func (s *ResendEmailSender) EnviarMagicLink(ctx context.Context, toEmail, link string) error {
	return s.enviar(ctx, toEmail, "Tu acceso a fudd",
		fmt.Sprintf("<p>Hola,</p><p>Haz clic en el siguiente enlace para ingresar a tu cuenta de fudd (expira en 15 minutos):</p><p><a href='%s'>Ingresar a fudd</a></p><p>Si no solicitaste este enlace, puedes ignorar este correo.</p>", link))
}

func (s *ResendEmailSender) EnviarInvitacion(ctx context.Context, toEmail, link string) error {
	return s.enviar(ctx, toEmail, "Te invitaron al equipo de fudd",
		fmt.Sprintf("<p>Hola,</p><p>Te invitaron a formar parte de un equipo en fudd.</p><p><a href='%s'>Aceptar invitación con Google</a></p><p>El enlace vence en 48 horas y puede usarse una sola vez. Debes continuar con la misma dirección de email que recibió esta invitación.</p>", link))
}

func (s *ResendEmailSender) enviar(ctx context.Context, toEmail, subject, htmlContent string) error {
	url := "https://api.resend.com/emails"
	body := map[string]any{
		"from":    s.fromEmail,
		"to":      []string{toEmail},
		"subject": subject,
		"html":    htmlContent,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("error serializando request de email: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error enviando request a Resend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("error de Resend: status %d", resp.StatusCode)
	}

	slog.InfoContext(ctx, "magic link enviado con éxito por Resend", "email", toEmail)
	return nil
}
