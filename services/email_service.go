package services

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"time"
)

// EmailService handles sending verification and password reset emails.
type EmailService interface {
	SendVerificationEmail(ctx context.Context, toEmail, token string) error
	SendPasswordResetEmail(ctx context.Context, toEmail, token string) error
}

type emailService struct {
	host     string
	port     int
	username string
	password string
	from     string
	baseURL  string
}

func NewEmailService(host string, port int, username, password, from, baseURL string) EmailService {
	return &emailService{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		baseURL:  baseURL,
	}
}

func (s *emailService) SendVerificationEmail(ctx context.Context, toEmail, token string) error {
	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", s.baseURL, token)

	subject := "Verify your YaneMarket account"
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: Arial, sans-serif; background-color: #1a1a2e; color: #ffffff; padding: 40px 20px;">
    <div style="max-width: 500px; margin: 0 auto; background: #2d2d44; border-radius: 16px; padding: 40px; box-shadow: 0 4px 20px rgba(0,0,0,0.3);">
        <div style="text-align: center; margin-bottom: 30px;">
            <h1 style="color: #d4af37; font-size: 28px; margin: 0;">YaneMarket</h1>
        </div>
        <h2 style="color: #ffffff; font-size: 22px; text-align: center;">Verify your email address</h2>
        <p style="color: #a0a0b0; font-size: 16px; line-height: 1.6; text-align: center;">
            Thank you for registering with YaneMarket. Please click the button below to verify your email address.
        </p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="%s" style="background: #d4af37; color: #1a1a2e; padding: 14px 40px; text-decoration: none; border-radius: 8px; font-weight: bold; font-size: 16px; display: inline-block;">Verify Email</a>
        </div>
        <p style="color: #a0a0b0; font-size: 14px; text-align: center;">
            If the button doesn't work, copy and paste this link into your browser:<br>
            <a href="%s" style="color: #d4af37; word-break: break-all;">%s</a>
        </p>
        <hr style="border: none; border-top: 1px solid #3d3d54; margin: 30px 0;">
        <p style="color: #606070; font-size: 12px; text-align: center;">
            This email was sent to %s. If you didn't create an account, you can safely ignore this email.
        </p>
    </div>
</body>
</html>`, verifyURL, verifyURL, verifyURL, toEmail)

	return s.sendEmail(toEmail, subject, body)
}

func (s *emailService) SendPasswordResetEmail(ctx context.Context, toEmail, token string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.baseURL, token)

	subject := "Reset your YaneMarket password"
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: Arial, sans-serif; background-color: #1a1a2e; color: #ffffff; padding: 40px 20px;">
    <div style="max-width: 500px; margin: 0 auto; background: #2d2d44; border-radius: 16px; padding: 40px; box-shadow: 0 4px 20px rgba(0,0,0,0.3);">
        <div style="text-align: center; margin-bottom: 30px;">
            <h1 style="color: #d4af37; font-size: 28px; margin: 0;">YaneMarket</h1>
        </div>
        <h2 style="color: #ffffff; font-size: 22px; text-align: center;">Reset your password</h2>
        <p style="color: #a0a0b0; font-size: 16px; line-height: 1.6; text-align: center;">
            You requested to reset your password. Click the button below to set a new password.
        </p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="%s" style="background: #d4af37; color: #1a1a2e; padding: 14px 40px; text-decoration: none; border-radius: 8px; font-weight: bold; font-size: 16px; display: inline-block;">Reset Password</a>
        </div>
        <p style="color: #a0a0b0; font-size: 14px; text-align: center;">
            If the button doesn't work, copy and paste this link into your browser:<br>
            <a href="%s" style="color: #d4af37; word-break: break-all;">%s</a>
        </p>
        <p style="color: #ff6b6b; font-size: 14px; text-align: center; margin-top: 20px;">
            This link will expire in 1 hour.
        </p>
        <hr style="border: none; border-top: 1px solid #3d3d54; margin: 30px 0;">
        <p style="color: #606070; font-size: 12px; text-align: center;">
            This email was sent to %s. If you didn't request a password reset, you can safely ignore this email.
        </p>
    </div>
</body>
</html>`, resetURL, resetURL, resetURL, toEmail)

	return s.sendEmail(toEmail, subject, body)
}

func (s *emailService) sendEmail(to, subject, htmlBody string) error {
	// MIME header for HTML email
	headers := fmt.Sprintf("MIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\nFrom: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n",
		s.from, to, subject)

	message := []byte(headers + htmlBody)

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	slog.Info("Sending email",
		slog.String("to", to),
		slog.String("subject", subject),
		slog.String("smtp_host", addr),
	)

	// Port 465 uses direct SSL, port 587 uses STARTTLS
	if s.port == 465 {
		// Direct SSL connection
		tlsConfig := &tls.Config{ServerName: s.host}
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, tlsConfig)
		if err != nil {
			slog.Error("Failed to connect to SMTP server (SSL)", slog.Any("error", err))
			return fmt.Errorf("failed to connect to SMTP server: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, s.host)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer client.Close()

		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
		if err := client.Mail(s.from); err != nil {
			return fmt.Errorf("failed to set sender: %w", err)
		}
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("failed to set recipient: %w", err)
		}
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("failed to send data: %w", err)
		}
		if _, err := w.Write(message); err != nil {
			return fmt.Errorf("failed to write message: %w", err)
		}
		if err := w.Close(); err != nil {
			return fmt.Errorf("failed to close data writer: %w", err)
		}
	} else {
		// STARTTLS (port 587)
		if err := smtp.SendMail(addr, auth, s.from, []string{to}, message); err != nil {
			slog.Error("Failed to send email", slog.Any("error", err))
			return fmt.Errorf("failed to send email: %w", err)
		}
	}

	slog.Info("Email sent successfully",
		slog.String("to", to),
		slog.String("subject", subject),
	)
	return nil
}
