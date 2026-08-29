package runtime

import (
	"errors"
	"fmt"
	"html/template"
	"net/url"
	"strings"
)

type emailContent struct {
	Subject string
	Text    string
	HTML    string
}

func renderIdentityEmail(kind, recipient, token, publicAppURL, verificationPath, resetPath string) (emailContent, error) {
	name := "there"
	if at := strings.IndexByte(recipient, '@'); at > 0 {
		name = recipient[:at]
	}
	if kind == "email-verification" {
		link := actionURL(publicAppURL, verificationPath, token)
		return emailContent{
			Subject: "Verify your e-commerce platform account",
			Text:    fmt.Sprintf("Hello %s,\n\nPlease verify your email address to activate your account:\n%s\n\nThis link expires in 24 hours and can be used only once. If you did not create this account, you can safely ignore this message.\n\n— The E-commerce Platform team\n", name, link),
			HTML:    renderHTML("Verify your account", "Confirm your email address to activate your E-commerce Platform account.", "Verify email address", link, "This link expires in 24 hours and can be used only once. If you did not create this account, you can safely ignore this message."),
		}, nil
	}
	if kind == "password-reset" {
		link := actionURL(publicAppURL, resetPath, token)
		return emailContent{
			Subject: "Reset your e-commerce platform password",
			Text:    fmt.Sprintf("Hello %s,\n\nWe received a request to reset your password:\n%s\n\nThis link expires in 30 minutes and can be used only once. If you did not request a reset, no action is required and your password remains unchanged.\n\n— The E-commerce Platform team\n", name, link),
			HTML:    renderHTML("Reset your password", "Use the secure link below to choose a new password.", "Reset password", link, "This link expires in 30 minutes and can be used only once. If you did not request a reset, no action is required."),
		}, nil
	}
	return emailContent{}, errors.New("unsupported identity email kind")
}

func actionURL(base, path, token string) string {
	parsed, err := url.Parse(strings.TrimRight(base, "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(path, "/") + "?token=" + url.QueryEscape(token)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/" + strings.TrimLeft(path, "/")
	query := parsed.Query()
	query.Set("token", token)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func renderHTML(title, intro, button, link, note string) string {
	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s</title></head>
<body style="margin:0;background:#f4f7fb;color:#172033;font-family:Arial,Helvetica,sans-serif;line-height:1.6">
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="background:#f4f7fb;padding:32px 12px"><tr><td align="center">
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="max-width:600px;background:#ffffff;border:1px solid #e6eaf0;border-radius:12px;overflow:hidden">
<tr><td style="background:#172033;padding:24px 32px;color:#ffffff;font-size:20px;font-weight:700">E-commerce Platform</td></tr>
<tr><td style="padding:36px 32px"><h1 style="margin:0 0 12px;font-size:28px;line-height:1.2;color:#172033">%s</h1><p style="margin:0 0 28px;color:#4a5568">%s</p>
<p style="margin:0 0 28px"><a href="%s" style="display:inline-block;background:#2563eb;color:#ffffff;text-decoration:none;border-radius:8px;padding:13px 20px;font-weight:700">%s</a></p>
<p style="margin:0;padding:16px;background:#f8fafc;border-radius:8px;color:#596579;font-size:14px">%s</p></td></tr>
<tr><td style="padding:20px 32px;border-top:1px solid #e6eaf0;color:#7a8494;font-size:12px">This is an automated message from the E-commerce Platform identity service. Please do not reply.</td></tr>
</table></td></tr></table></body></html>`, template.HTMLEscapeString(title), template.HTMLEscapeString(title), template.HTMLEscapeString(intro), template.HTMLEscapeString(link), template.HTMLEscapeString(button), template.HTMLEscapeString(note))
}
