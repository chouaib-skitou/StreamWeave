package runtime

import (
	_ "embed"
	"errors"
	"html/template"
	"net/url"
	"strings"
)

//go:embed assets/platform-logo.png
var platformLogo []byte

type emailContent struct {
	Subject string
	Text    string
	HTML    string
}

func renderIdentityEmail(kind, recipient, token, publicAppURL, verificationPath, resetPath string) (emailContent, error) {
	return renderIdentityEmailWithSupport(kind, recipient, token, publicAppURL, verificationPath, resetPath, "support@example.com")
}

func renderIdentityEmailWithSupport(kind, recipient, token, publicAppURL, verificationPath, resetPath, supportEmail string) (emailContent, error) {
	name := "there"
	if at := strings.IndexByte(recipient, '@'); at > 0 {
		name = recipient[:at]
	}
	if kind == "email-verification" {
		link := actionURL(publicAppURL, verificationPath, token)
		return emailContent{
			Subject: "Verify your Event Commerce Platform account",
			Text:    renderText(name, "Verify your account", "Confirm your email address to activate your Event Commerce Platform account.", link, "This link expires in 24 hours and can be used only once. If you did not create this account, you can safely ignore this message.", supportEmail),
			HTML:    renderHTML(name, "Verify your account", "Confirm your email address to activate your Event Commerce Platform account.", "Verify email address", link, "This link expires in 24 hours and can be used only once. If you did not create this account, you can safely ignore this message.", supportEmail),
		}, nil
	}
	if kind == "password-reset" {
		link := actionURL(publicAppURL, resetPath, token)
		return emailContent{
			Subject: "Reset your Event Commerce Platform password",
			Text:    renderText(name, "Reset your password", "Use the secure link below to choose a new password.", link, "This link expires in 30 minutes and can be used only once. If you did not request a reset, no action is required.", supportEmail),
			HTML:    renderHTML(name, "Reset your password", "Use the secure link below to choose a new password.", "Reset password", link, "This link expires in 30 minutes and can be used only once. If you did not request a reset, no action is required.", supportEmail),
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

func renderText(name, title, intro, link, note, supportEmail string) string {
	return strings.Join([]string{
		"Event Commerce Platform",
		"",
		"Hello " + name + ",",
		"",
		title,
		intro,
		"",
		link,
		"",
		note,
		"",
		"Need help? Contact " + supportEmail + ".",
		"",
		"© 2026 Event Commerce Platform · This is an automated message.",
		"",
	}, "\n")
}

func renderHTML(name, title, intro, button, link, note, supportEmail string) string {
	return `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + template.HTMLEscapeString(title) + `</title></head>
<body style="margin:0;background:#f5f7fb;color:#111827;font-family:Inter,Segoe UI,Arial,sans-serif;line-height:1.6">
<div style="display:none;max-height:0;overflow:hidden;opacity:0">` + template.HTMLEscapeString(intro) + `</div>
<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background:#f5f7fb;padding:40px 16px"><tr><td align="center">
<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="max-width:620px;background:#ffffff;border:1px solid #e5e7eb;border-radius:20px;overflow:hidden;box-shadow:0 12px 32px rgba(15,23,42,.08)">
<tr><td style="background:#0b1020;padding:28px 36px"><table role="presentation" cellspacing="0" cellpadding="0"><tr><td style="vertical-align:middle"><img src="cid:platform-logo" width="52" height="52" alt="Event Commerce Platform" style="display:block;border:0;border-radius:14px"></td><td style="padding-left:14px;vertical-align:middle;color:#ffffff"><div style="font-size:18px;font-weight:700;letter-spacing:-.2px">Event Commerce</div><div style="font-size:12px;color:#9eeeff;letter-spacing:1.5px;text-transform:uppercase">Platform</div></td></tr></table></td></tr>
<tr><td style="padding:44px 36px 36px"><p style="margin:0 0 10px;color:#64748b;font-size:14px">Hello ` + template.HTMLEscapeString(name) + `,</p><h1 style="margin:0 0 14px;font-size:32px;line-height:1.15;letter-spacing:-.8px;color:#0f172a">` + template.HTMLEscapeString(title) + `</h1><p style="margin:0 0 30px;color:#475569;font-size:16px">` + template.HTMLEscapeString(intro) + `</p>
<p style="margin:0 0 30px"><a href="` + template.HTMLEscapeString(link) + `" style="display:inline-block;background:#06b6d4;color:#06202b;text-decoration:none;border-radius:10px;padding:14px 22px;font-weight:800;box-shadow:0 6px 16px rgba(6,182,212,.25)">` + template.HTMLEscapeString(button) + ` &nbsp;→</a></p>
<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:12px"><tr><td style="padding:18px 20px;color:#475569;font-size:14px"><strong style="color:#1e293b">Security note</strong><br>` + template.HTMLEscapeString(note) + `</td></tr></table></td></tr>
<tr><td style="padding:24px 36px 30px;border-top:1px solid #eef2f7;color:#64748b;font-size:12px;line-height:1.7">Need help? Contact <a href="mailto:` + template.HTMLEscapeString(supportEmail) + `" style="color:#0891b2;text-decoration:none">` + template.HTMLEscapeString(supportEmail) + `</a>.<br>This is an automated security message from Event Commerce Platform. Please do not reply.<br><span style="color:#94a3b8">© 2026 Event Commerce Platform</span></td></tr>
</table></td></tr></table></body></html>`
}
