# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability in pyre, please report it responsibly:

1. **Do not** open a public issue
2. Email details to the repository owner via GitHub private message
3. Include steps to reproduce and potential impact

We will respond within 72 hours and work with you to address the issue.

## Security Best Practices

### API Key Management

pyre handles sensitive Palo Alto firewall API keys. Follow these practices:

**Recommended:**
- Use environment variables for API keys (`api_key_env` in config)
- Set restrictive permissions on `~/.pyre.yaml` (`chmod 600`)
- Use separate API keys per firewall with minimal required permissions
- Rotate API keys periodically

**Avoid:**
- Storing API keys directly in configuration files
- Committing credentials to version control
- Sharing API keys between users

### Configuration Example

```yaml
firewalls:
  prod-fw01:
    host: firewall.example.com
    api_key_env: PROD_FW01_API_KEY  # Read from environment
    insecure: false                  # Verify TLS certificates
```

### TLS Certificate Verification

- Keep `insecure: false` (default) for production firewalls
- Only use `insecure: true` for development/lab environments with self-signed certificates
- Consider importing firewall certificates to your trust store instead of disabling verification

### Network Security

- pyre communicates with firewalls over HTTPS (port 443) and SSH (port 22)
- Ensure network policies allow only authorized hosts to connect to firewall management interfaces
- Consider using jump hosts or VPNs for remote management

## Dependencies

pyre uses these third-party libraries:

- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/bubbles` - TUI components
- `github.com/charmbracelet/lipgloss` - Styling
- `gopkg.in/yaml.v3` - YAML parsing

We monitor dependencies for vulnerabilities and update promptly when issues are disclosed.

## Audit Logging

pyre does not maintain its own audit logs. Firewall API calls are logged by PAN-OS according to your firewall's logging configuration. Review firewall logs to audit pyre usage.
