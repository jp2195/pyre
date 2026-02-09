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
- Use environment variables for API keys (`PYRE_API_KEY`)
- Use the `--api-key` flag for one-off connections
- Use separate API keys per firewall with minimal required permissions
- Rotate API keys periodically

**Avoid:**
- Storing API keys in configuration files (not supported)
- Committing credentials to version control
- Sharing API keys between users

### Configuration Example

```yaml
connections:
  10.0.0.1:
    insecure: false  # Verify TLS certificates
```

Then provide the API key via environment:

```bash
export PYRE_API_KEY=YOUR_API_KEY
pyre -c 10.0.0.1
```

### TLS Certificate Verification

- Keep `insecure: false` (default) for production firewalls
- Only use `insecure: true` for development/lab environments with self-signed certificates
- Consider importing firewall certificates to your trust store instead of disabling verification

### Network Security

- pyre communicates with firewalls over HTTPS (port 443)
- Ensure network policies allow only authorized hosts to connect to firewall management interfaces
- Consider using jump hosts or VPNs for remote management

## Dependencies

pyre uses these third-party libraries:

- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/bubbles` - TUI components
- `github.com/charmbracelet/lipgloss` - Styling
- `go.yaml.in/yaml/v4` - YAML parsing

We monitor dependencies for vulnerabilities and update promptly when issues are disclosed.

## Audit Logging

pyre does not maintain its own audit logs. Firewall API calls are logged by PAN-OS according to your firewall's logging configuration. Review firewall logs to audit pyre usage.


