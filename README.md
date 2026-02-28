# GCP SSH-in-Browser Launcher (Created by Claude)

A simple Go CLI tool that skips all the manual steps of navigating GCP Console and directly opens **SSH-in-Browser** for your Compute Engine instances.

## How It Works

GCP has a direct URL for SSH-in-browser:
```
https://ssh.cloud.google.com/projects/{PROJECT}/zones/{ZONE}/instances/{INSTANCE}
```

This tool constructs that URL and opens Chrome **with your specific profile** (so your Google account is already authenticated). No extra login needed!

## Setup

### 1. Build the binary

```bash
cd gcp-ssh
go build -o gcp-ssh .
```

### 2. (Optional) Add to PATH

```bash
# Linux/macOS
sudo mv gcp-ssh /usr/local/bin/

# Or add to your shell profile
export PATH="$PATH:/path/to/gcp-ssh"
```

### 3. First run — set your Chrome profile

```bash
gcp-ssh
```

It will auto-discover your Chrome profiles and ask you to pick one.
Choose the profile where you're logged into your GCP account.

## Usage

### Interactive Mode
```bash
gcp-ssh
```
Gives you a menu to connect, add, list, or remove instances.

### Quick Connect (one-off)
```bash
gcp-ssh quick my-project-id us-central1-a my-instance-name
```

### Save & Connect by Alias
```bash
# Save an instance
gcp-ssh add

# Connect using the alias
gcp-ssh my-server

# Or explicitly
gcp-ssh connect my-server
```

### Other Commands
```bash
gcp-ssh list              # List all saved instances
gcp-ssh remove my-server  # Remove a saved instance
gcp-ssh profile           # Change Chrome profile
gcp-ssh help              # Show help
```

## Example Workflow

```
$ gcp-ssh quick my-gcp-project asia-south1-a dev-machine

  🚀 Opening SSH for: dev-machine (zone: asia-south1-a, project: my-gcp-project)
  🌐 URL: https://ssh.cloud.google.com/projects/my-gcp-project/zones/asia-south1-a/instances/dev-machine?authuser=0
  🔑 Chrome profile: Profile 1
  ✓ Chrome launched! SSH session will authenticate automatically.
```

## Config

All configuration is stored at `~/.gcp-ssh/config.json`:

```json
{
  "chrome_profile_dir": "Profile 1",
  "instances": [
    {
      "alias": "dev",
      "project": "my-project-id",
      "zone": "us-central1-a",
      "name": "dev-instance",
      "authuser": 0
    }
  ]
}
```

You can also edit this file directly.

## Notes

- **authuser**: If you have multiple Google accounts in Chrome, `0` is the first, `1` is the second, etc. Match it to whichever account has GCP access.
- **Cross-platform**: Works on macOS, Linux, and Windows.
- **No dependencies**: Pure Go standard library, no external packages.
- **Chrome must be installed**: Falls back to the default browser if Chrome isn't found.

## Finding Your Chrome Profile

If auto-discovery doesn't work, you can find your profile manually:

- **macOS**: `~/Library/Application Support/Google/Chrome/`
- **Linux**: `~/.config/google-chrome/`
- **Windows**: `%LOCALAPPDATA%\Google\Chrome\User Data\`

Look for directories named `Default`, `Profile 1`, `Profile 2`, etc.
Open `chrome://version` in your browser to see which profile directory is active.
