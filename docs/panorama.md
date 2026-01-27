# Panorama Support

pyre can connect to Panorama and manage its managed firewalls through device targeting.

## Connecting to Panorama

Configure Panorama in your config file:

```yaml
connections:
  panorama.example.com:
    type: panorama
    insecure: true
```

When you connect, pyre automatically detects Panorama and loads the list of managed devices.

## Device Picker

Once connected to Panorama, press `D` to open the device picker. This shows all managed firewalls with:

| Column | Description |
|--------|-------------|
| Hostname | Device hostname |
| Serial | Serial number |
| Model | Hardware model (PA-3260, VM-100, etc.) |
| HA State | Active, Passive, or Standalone |
| Connected | Whether Panorama can reach the device |
| IP Address | Management IP |

### Selecting a Device

1. Navigate with `j`/`k` or arrow keys
2. Press `Enter` to select

Once selected, all commands are proxied through Panorama to the target device. The status bar shows the current target.

### Targeting Panorama Directly

Select "Panorama" in the device picker to run commands directly on Panorama rather than on a managed firewall.

## Status Bar Indicator

The header shows your current target:

```
● panorama (panorama.example.com) → fw-dc1-01
```

- Connection name and host
- Arrow (`→`) showing the targeted device
- Target device hostname or serial

When targeting Panorama directly:
```
● panorama (panorama.example.com)
```

## What Gets Proxied

When targeting a managed firewall, these operations are proxied:

- System information
- Resource utilization
- Interface status
- Security and NAT policies
- Active sessions
- Logs
- HA status

Some Panorama-specific operations run directly on Panorama regardless of target:
- Managed device list
- Template and device group configuration

## Refreshing the Device List

In the device picker, press `r` to refresh the list of managed devices. This is useful when:
- A new device was added to Panorama
- A device's connection status changed
- HA state changed

## Configuration Tips

### Panorama and Direct Firewall Entries

You can have both Panorama and direct firewall entries:

```yaml
connections:
  panorama.example.com:
    type: panorama
    insecure: true

  # Direct entries for individual firewalls
  10.1.0.1:
    insecure: true
```

This gives you flexibility to connect directly to firewalls or through Panorama.

### API Key Permissions

The Panorama API key needs appropriate permissions:
- Read access to managed devices
- Operational command permissions for targeted operations
- Configuration read access for policies

The same permissions needed on standalone firewalls apply when proxying through Panorama.

## Troubleshooting

### Device Shows "Disconnected"

If a device shows as disconnected:
- Check the device's connectivity to Panorama
- Verify the device is properly registered in Panorama
- Check for certificate or licensing issues

### Commands Fail on Target

If commands work on Panorama but fail on a target:
- The target device may have restricted API access
- Template permissions may be limiting operations
- The device may be unreachable from Panorama

### Slow Response Times

Proxied commands take longer than direct connections because:
- Request goes: pyre -> Panorama -> Target -> Panorama -> pyre
- Each hop adds latency

For performance-critical operations, consider connecting directly to the firewall.
