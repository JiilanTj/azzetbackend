package shared

import (
	"context"
	"fmt"
	"log/slog"

	smtpclient "codeberg.org/azzet/azzetbe/internal/smtp"
)

// EmailOTPSender sends OTP codes via SMTP email.
type EmailOTPSender struct {
	Host string
	Port string
	User string
	Pass string
	From string
	Env  string
}

// NewEmailOTPSender constructs an EmailOTPSender with the given SMTP credentials.
func NewEmailOTPSender(host, port, user, pass, from, env string) *EmailOTPSender {
	return &EmailOTPSender{
		Host: host,
		Port: port,
		User: user,
		Pass: pass,
		From: from,
		Env:  env,
	}
}

// SendOTP sends a one-time password to the given email address.
// If SMTP is not configured it silently returns nil (with a debug log in development).
func (e *EmailOTPSender) SendOTP(_ context.Context, to, code string) error {
	if e.Host == "" || e.User == "" {
		if e.Env == "development" {
			slog.Debug("email OTP generated (SMTP not configured)", "to", to)
			return nil
		}
		return fmt.Errorf("email delivery is not configured")
	}

	mailer := smtpclient.New(e.Host, e.Port, e.User, e.Pass, e.From)

	html := buildOTPEmailHTML(code)

	if err := mailer.Send(smtpclient.Email{
		To:       []string{to},
		Subject:  "Kode Verifikasi Azzet",
		HTMLBody: html,
	}); err != nil {
		slog.Error("email OTP: gagal mengirim", "to", to, "error", err)
		return fmt.Errorf("email OTP: gagal mengirim: %w", err)
	}

	slog.Info("email OTP terkirim", "to", to)
	return nil
}

// buildOTPEmailHTML returns a styled HTML email body containing the OTP code.
func buildOTPEmailHTML(code string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Kode Verifikasi Azzet</title>
</head>
<body style="margin:0;padding:0;background-color:#f4f6f9;font-family:'Segoe UI',Arial,sans-serif;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f4f6f9;padding:40px 0;">
    <tr>
      <td align="center">
        <table width="560" cellpadding="0" cellspacing="0"
               style="background:#ffffff;border-radius:12px;overflow:hidden;
                      box-shadow:0 2px 8px rgba(0,0,0,.08);">

          <!-- Header -->
          <tr>
            <td align="center"
                style="background:linear-gradient(135deg,#2563eb 0%%,#1d4ed8 100%%);
                       padding:36px 40px;">
              <span style="font-size:26px;font-weight:700;color:#ffffff;
                           letter-spacing:-0.5px;">Azzet</span>
            </td>
          </tr>

          <!-- Body -->
          <tr>
            <td style="padding:40px 48px 32px;">
              <p style="margin:0 0 8px;font-size:22px;font-weight:600;
                        color:#111827;">Kode Verifikasi Anda</p>
              <p style="margin:0 0 28px;font-size:15px;color:#6b7280;
                        line-height:1.6;">
                Gunakan kode di bawah ini untuk melanjutkan masuk ke akun Azzet Anda.
                Kode ini hanya berlaku selama <strong>5 menit</strong> dan
                <strong>tidak boleh dibagikan</strong> kepada siapa pun.
              </p>

              <!-- OTP box -->
              <table width="100%%" cellpadding="0" cellspacing="0"
                     style="margin-bottom:28px;">
                <tr>
                  <td align="center"
                      style="background:#eff6ff;border:2px dashed #93c5fd;
                             border-radius:10px;padding:24px;">
                    <span style="font-size:40px;font-weight:700;
                                 letter-spacing:12px;color:#1d4ed8;
                                 font-family:'Courier New',monospace;">%s</span>
                  </td>
                </tr>
              </table>

              <p style="margin:0 0 24px;font-size:14px;color:#9ca3af;
                        line-height:1.6;">
                Jika Anda tidak meminta kode ini, abaikan email ini.
                Akun Anda tetap aman dan tidak ada tindakan yang diperlukan.
              </p>

              <hr style="border:none;border-top:1px solid #e5e7eb;margin:0 0 24px;" />

              <p style="margin:0;font-size:13px;color:#d1d5db;text-align:center;">
                &copy; Azzet &mdash; Platform Akuntansi &amp; Keuangan Enterprise
              </p>
            </td>
          </tr>

        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, code)
}
