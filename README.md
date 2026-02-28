# GCP SSH Launcher

A Go CLI that connects to Google Compute Engine instances in either:

- **Browser SSH** (`ssh.cloud.google.com`), or
- **Terminal SSH** (`gcloud compute ssh`).

Before connecting, it integrates with **gcloud CLI** to:

1. Verify `gcloud` is installed,
2. Ensure the active account matches the configured account (if provided),
3. Check VM status and start it if it is not running.

If `gcloud` is missing, the tool prompts you to start the instance manually.

## Build

```bash
go build -o gcp-ssh .
```

## Usage

### Interactive mode

```bash
gcp-ssh
```

### Browser SSH (one-off)

```bash
gcp-ssh quick <project> <zone> <instance> [gcloud-account-email]
```

### Terminal SSH (one-off)

```bash
gcp-ssh quick-terminal <project> <zone> <instance> [gcloud-account-email]
```

### Saved instances

```bash
gcp-ssh add
gcp-ssh list
gcp-ssh connect <alias>
gcp-ssh connect-terminal <alias>
gcp-ssh remove <alias>
```

### Profile/help

```bash
gcp-ssh profile
gcp-ssh help
```

## Config

Configuration is stored in `~/.gcp-ssh/config.json`.

Example:

```json
{
  "chrome_profile_dir": "Profile 1",
  "instances": [
    {
      "alias": "dev",
      "project": "my-project-id",
      "zone": "us-central1-a",
      "name": "dev-instance",
      "authuser": 0,
      "gcloud_account": "user@example.com",
      "connection_mode": "browser"
    }
  ]
}
```

## Notes

- `authuser` controls which Google account is used in browser SSH URL (`0`, `1`, etc.).
- `gcloud_account` is used to verify/switch/login in gcloud before VM start/SSH.
- `connection_mode` can be `browser` or `terminal` for saved instances.
