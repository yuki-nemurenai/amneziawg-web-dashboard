# AmneziaWG Dashboard and Management Panel

A web-based dashboard and management panel for AmneziaWG (AWG) servers.

![Go](https://img.shields.io/badge/Backend-Golang_1.26-00ADD8?style=flat&logo=go)
![React](https://img.shields.io/badge/Frontend-React_19-61DAFB?style=flat&logo=react)
![PostgreSQL](https://img.shields.io/badge/Database-PostgreSQL_18-4169E1?style=flat&logo=postgresql)
![Docker](https://img.shields.io/badge/Deployment-Docker_Compose-2496ED?style=flat&logo=docker)


## Features

- **User Authentication & First-Run Wizard**: Secure first-run setup wizard for creating the initial user, bcrypt password hashing, and JWT-based authentication.
- **Real-Time Traffic & Connection Monitoring**: Live bandwidth statistics (`Rx/Tx bytes`), latest handshake times, and online/offline status indicators.
- **QR Codes & Client Configuration Pairing**: One-click generation of client `.conf` files and instant QR code display for easy pairing with mobile devices using the AmneziaWG and AmneziaVPN apps.
- **AmneziaWG ASC Parameter Manager**: GUI editor for server settings and anti-censorship parameters, including `Jc`, `Jmin`, `Jmax`, `S1`–`S4`, `H1`–`H4`, and `I1`–`I4`.
- **Dynamic Interface Synchronization**: Adds and removes peers dynamically using `awg syncconf` without interrupting existing active client tunnels.
- **Containerized Architecture**: Separate Docker containers for the React frontend (served by Nginx), Go REST API backend, and PostgreSQL database.

## Requirements

- A Linux VM or server with at least 1 CPU core and 1 GB of RAM
- Docker Engine and Docker Compose

## Installation

### Upgrade Your System (Optional)

Upgrade your system packages and kernel:

```bash
sudo apt update && sudo apt upgrade -y
```

If a new kernel is installed, reboot the system before continuing.

### Install Docker Engine

Install the latest Docker Engine:

```bash
curl -fsSL https://get.docker.com | bash
sudo usermod -aG docker $USER
```

Log out and log back in for the new group membership to take effect.

## Configure IP Forwarding

Enable IPv4 and IPv6 forwarding:

```bash
# Runtime
sudo sysctl -w net.ipv4.ip_forward=1
sudo sysctl -w net.ipv6.conf.all.forwarding=1

# Persistent
echo "net.ipv4.ip_forward = 1" | sudo tee /etc/sysctl.d/99-forwarding.conf
echo "net.ipv6.conf.all.forwarding = 1" | sudo tee -a /etc/sysctl.d/99-forwarding.conf
```

## Network & Kernel Tuning (Optional)

The following `sysctl` parameters can be used to tune TCP buffers and the Linux network stack.

Create `/etc/sysctl.d/99-network-tune.conf`:

```ini
# Fair Queueing
net.core.default_qdisc = fq

# TCP Congestion Control
net.ipv4.tcp_congestion_control = bbr

# TCP Buffer Optimization
net.core.rmem_max = 134217728
net.core.wmem_max = 134217728
net.ipv4.tcp_rmem = 4096 87380 134217728
net.ipv4.tcp_wmem = 4096 65536 134217728

# TCP Performance Tuning
net.ipv4.tcp_window_scaling = 1
net.ipv4.tcp_timestamps = 1
net.ipv4.tcp_sack = 1
net.ipv4.tcp_no_metrics_save = 1
net.ipv4.tcp_moderate_rcvbuf = 1

# Network Stack Optimization
net.core.netdev_max_backlog = 5000
net.ipv4.tcp_slow_start_after_idle = 0
```

Apply the changes:

```bash
sudo sysctl --system
```

Verify that BBR is available and enabled:

```bash
sysctl net.ipv4.tcp_congestion_control
```

Expected output:

```text
net.ipv4.tcp_congestion_control = bbr
```

> **Note:** The `bbr` setting enables the BBR implementation provided by your kernel. It does not by itself mean that BBRv3 is installed or enabled.

## Install the AmneziaWG Kernel Module (Optional, but Strongly Recommended)

The AmneziaWG kernel module is required to run an AmneziaWG interface using the kernel implementation.

### Install Build Dependencies

```bash
sudo apt install -y \
    git \
    gcc \
    make \
    linux-headers-$(uname -r)
```

### Clone the AmneziaWG Kernel Module Repository

```bash
git clone https://github.com/amnezia-vpn/amneziawg-linux-kernel-module.git
cd amneziawg-linux-kernel-module/src
```

For modern Linux kernels (`5.6+`), the AmneziaWG build process requires access to the full kernel source tree. Create a `kernel` symlink pointing to the appropriate kernel source tree:

```bash
ln -s /path/to/kernel/source kernel
```

Then build and install the module:

```bash
make
sudo make install
```

Load the kernel module:

```bash
sudo modprobe amneziawg
```

Verify that the module is loaded:

```bash
lsmod | grep amneziawg
```

Example output:

```text
amneziawg             135168  0
libcurve25519          65536  1 amneziawg
ip6_udp_tunnel         16384  1 amneziawg
udp_tunnel             36864  1 amneziawg
```

For more information about supported installation methods and kernel requirements, see the [official AmneziaWG kernel module documentation](https://github.com/amnezia-vpn/amneziawg-linux-kernel-module).

## Install the AmneziaWG Web Dashboard

Clone the repository:

```bash
git clone https://github.com/yuki-nemurenai/amneziawg-web-dashboard
cd awg-web
```

Copy the example environment file:

```bash
cp .env.example .env
```

Edit the `.env` file as needed:

```ini
POSTGRES_USER=awg
POSTGRES_PASSWORD=awgsecretpassword
POSTGRES_DB=awg_db
JWT_SECRET=super-secret-jwt-key-replace-in-production
AWG_PORT=8443
```

> **Important:** Replace `POSTGRES_PASSWORD` and `JWT_SECRET` with strong, unique values before deploying to production.

Start the dashboard:

```bash
docker compose up -d
```

Open the dashboard at:

```text
http://localhost
```

or:

```text
http://<your-server-ip>
```

On first startup, the **Initial User Setup Wizard** will prompt you to create an administrator account. After registration, you will be redirected to the **AmneziaWG Server Control Panel**.

## Architecture

```text
                  ┌─────────────────────────────────┐
                  │          Client Browser         │
                  └────────────────┬────────────────┘
                                   │ HTTP :80
                                   ▼
                 ┌───────────────────────────────────┐
                 │ awg-frontend (Nginx)              │
                 │ Serves the React SPA               │
                 │ Proxies /api/* to awg-backend      │
                 └─────────────────┬─────────────────┘
                                   │ HTTP :8080
                                   ▼
                 ┌───────────────────────────────────┐
                 │ awg-backend (Go API)              │
                 │ REST API + JWT authentication     │
                 └────────┬──────────────────┬───────┘
                          │                  │
                 Reads/Writes               │ SQL queries
                          │                  │
                          ▼                  ▼
             ┌──────────────────────┐   ┌──────────────────────┐
             │ awg0.conf & AWG       │   │ awg-db               │
             │ host interface        │   │ PostgreSQL           │
             └──────────────────────┘   └──────────────────────┘
```

## Tech Stack

- **Backend**: Go 1.26, `go-chi/v5`, `jackc/pgx/v5`, `golang-jwt/jwt/v5`, `golang.org/x/crypto`, `log/slog`
- **Frontend**: React 19, Vite, Tailwind CSS, Lucide Icons
- **Database**: PostgreSQL 18
- **Web Server**: Nginx

## License

[MIT](LICENSE)

## Buy Me A Coffee
If you find this project useful, consider supporting its development with a donation via Boosty. Your support helps keep the project going and makes future improvements possible. Thank you! ❤️

[!["Buy Me A Coffee"](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://boosty.to/yuki_nemurenai/donate)
