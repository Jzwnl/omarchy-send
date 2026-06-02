#!/usr/bin/env bash
#
# install.sh — install Omarchy-Send.
#
# Quick install (nothing to clone — the Once way):
#
#   curl -fsSL https://raw.githubusercontent.com/28allday/omarchy-send/main/install.sh | bash
#
# When run from a git clone it builds from source instead (if Go is present),
# otherwise it downloads the latest released binary for your architecture.
#
#   ./install.sh
#
# Environment overrides:
#   BIN_DIR=/usr/local/bin           install location (default ~/.local/bin)
#   OMARCHY_SEND_VERSION=v0.1.0       pin a release (default: latest)
#   OMARCHY_SEND_MODE=local|remote    skip the local/remote prompt (default: ask,
#                                     or local when non-interactive)
#
# Behaviour:
#   - Local machine (home/LAN): installs the TUI; on Omarchy also adds a Walker
#     entry + the Nautilus right-click integration.
#   - Remote server (public IP): same install, then restricts port 53317 to the
#     Tailscale interface in the firewall, so it's reachable over the tailnet
#     only — not the open internet.

set -euo pipefail

REPO="28allday/omarchy-send"
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"
APP_DIR="$HOME/.local/share/applications"
BIN="$BIN_DIR/omarchy-send"
VERSION="${OMARCHY_SEND_VERSION:-latest}"
PORT=53317

mkdir -p "$BIN_DIR"

# If the script lives next to the source tree, we're in a clone.
SCRIPT_DIR=""
if [ -n "${BASH_SOURCE[0]:-}" ] && [ -f "${BASH_SOURCE[0]}" ]; then
  SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi

# ---- local vs remote -----------------------------------------------------
# A remote (public-IP) server should not expose the transfer port to the
# internet. Ask once; default to "local" when non-interactive (e.g. piped
# `curl | bash` with no terminal) so a firewall is never changed without intent.
# Reads /dev/tty so the prompt still works under curl|bash.
MODE="${OMARCHY_SEND_MODE:-}"
case "$MODE" in
  local | remote) : ;; # explicit override, don't ask
  *)
    MODE="local"
    # Try to open the controlling terminal read-write on fd 3. A bare -r test
    # isn't enough: /dev/tty can exist yet fail to open (no controlling tty —
    # cron/CI/piped). Only prompt when the open actually succeeds.
    if { exec 3<>/dev/tty; } 2>/dev/null; then
      printf 'Install type — [L]ocal machine (home/LAN) or [r]emote server (public IP)? [L/r] ' >&3 || true
      IFS= read -r _ans <&3 || _ans=""
      exec 3>&- 3<&- || true
      case "$_ans" in
        r | R | remote | Remote | REMOTE) MODE="remote" ;;
      esac
    fi
    ;;
esac

# ---- obtain the binary ---------------------------------------------------
if [ -n "$SCRIPT_DIR" ] && [ -f "$SCRIPT_DIR/go.mod" ] && command -v go >/dev/null 2>&1; then
  echo "==> Building omarchy-send from source..."
  (cd "$SCRIPT_DIR" && go build -o "$BIN" ./cmd/omarchy-send)
else
  # Download the released binary for this OS/arch (curl-style install).
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  if [ "$os" != "linux" ]; then
    echo "ERROR: omarchy-send ships Linux binaries only (detected: $os)." >&2
    echo "       On other systems, clone the repo and build with Go." >&2
    exit 1
  fi
  case "$(uname -m)" in
    x86_64 | amd64) arch=amd64 ;;
    aarch64 | arm64) arch=arm64 ;;
    *) echo "ERROR: unsupported architecture: $(uname -m)" >&2; exit 1 ;;
  esac
  asset="omarchy-send-${os}-${arch}"
  if [ "$VERSION" = "latest" ]; then
    url="https://github.com/$REPO/releases/latest/download/$asset"
  else
    url="https://github.com/$REPO/releases/download/$VERSION/$asset"
  fi

  echo "==> Downloading $asset ($VERSION)..."
  tmp="$(mktemp)"
  trap 'rm -f "$tmp"' EXIT
  if command -v curl >/dev/null 2>&1; then
    curl -fSL --proto '=https' --tlsv1.2 -o "$tmp" "$url"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$tmp" "$url"
  else
    echo "ERROR: need curl or wget to download the binary." >&2
    exit 1
  fi
  install -m 755 "$tmp" "$BIN"
fi
echo "    Installed: $BIN"

# ---- Omarchy desktop integration -----------------------------------------
# Only on Omarchy: add a Walker entry that opens the TUI in a floating window.
# The TUI.float app-id is matched by Omarchy's stock floating-window rule, so
# no Hyprland configuration is required.
if command -v omarchy-launch-tui >/dev/null 2>&1 || [ -d "$HOME/.local/share/omarchy" ]; then
  mkdir -p "$APP_DIR"

  # Install the bundled icon into the user's hicolor theme. The SVG is embedded
  # so the curl-piped install has nothing extra to fetch.
  icon_dir="$HOME/.local/share/icons/hicolor/scalable/apps"
  mkdir -p "$icon_dir"
  cat > "$icon_dir/omarchy-send.svg" <<'SVG'
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" width="256" height="256">
  <rect width="256" height="256" rx="56" fill="#16161e"/>
  <!-- paper-plane: upper wing (light) + lower flap (darker) -->
  <path d="M214 42 L42 114 L110 142 Z" fill="#7aa2f7"/>
  <path d="M214 42 L110 142 L130 214 L158 166 Z" fill="#5a7fd6"/>
</svg>
SVG
  gtk-update-icon-cache -q -t -f "$HOME/.local/share/icons/hicolor" 2>/dev/null || true

  cat > "$APP_DIR/omarchy-send.desktop" <<EOF
[Desktop Entry]
Name=Omarchy-Send
Comment=Send & receive files over the LAN (LocalSend-compatible)
Exec=xdg-terminal-exec --app-id=TUI.float -e $BIN
Icon=omarchy-send
Terminal=false
Type=Application
Categories=Network;FileTransfer;
Keywords=localsend;share;transfer;airdrop;
EOF
  echo "==> Omarchy detected — added floating Walker entry (with icon)."
  echo "    Launch it from Walker by searching 'Omarchy-Send'."

  # Nautilus right-click: "Send via Omarchy-Send". Same approach as Omarchy's
  # Transcode entry — a nautilus-python MenuProvider that opens the TUI in a
  # floating presentation terminal with the selected paths pre-staged. Installed
  # from the clone when present, otherwise written from an embedded copy so the
  # curl-piped install needs nothing extra.
  #
  # Desktop-only: skip entirely when Nautilus isn't installed (e.g. a headless
  # server). The right-click flow is a graphical convenience and is not required
  # there — the TUI and headless send still work without it.
  if command -v nautilus >/dev/null 2>&1; then
  ext_dir="$HOME/.local/share/nautilus-python/extensions"
  mkdir -p "$ext_dir"
  if [ -n "$SCRIPT_DIR" ] && [ -f "$SCRIPT_DIR/nautilus/omarchy-send.py" ]; then
    cp "$SCRIPT_DIR/nautilus/omarchy-send.py" "$ext_dir/omarchy-send.py"
  else
    cat > "$ext_dir/omarchy-send.py" <<'PY'
import os
import shlex
import shutil

from gi import require_version

require_version("Nautilus", "4.1")

from gi.repository import GObject, Gio, Nautilus


# omarchy-send installs to ~/.local/bin, which is often NOT on the PATH the
# Nautilus process (and the terminal it spawns) inherits from the graphical
# session — so resolve it to an absolute path and invoke it by that.
def _resolve(name, fallbacks):
    found = shutil.which(name)
    if found:
        return found
    for path in fallbacks:
        if path and os.path.isfile(path) and os.access(path, os.X_OK):
            return path
    return None


def _binary():
    home = os.path.expanduser("~")
    fallbacks = []
    bin_dir = os.environ.get("BIN_DIR")
    if bin_dir:
        fallbacks.append(os.path.join(bin_dir, "omarchy-send"))
    fallbacks.append(os.path.join(home, ".local", "bin", "omarchy-send"))
    fallbacks.append(os.path.join(home, "bin", "omarchy-send"))
    return _resolve("omarchy-send", fallbacks)


def _wrapper():
    home = os.path.expanduser("~")
    fallbacks = [
        os.path.join(home, ".local", "share", "omarchy", "bin",
                     "omarchy-launch-floating-terminal-with-presentation"),
    ]
    return _resolve("omarchy-launch-floating-terminal-with-presentation", fallbacks)


class OmarchySendAction(GObject.GObject, Nautilus.MenuProvider):
    def _launch(self, paths):
        wrapper = _wrapper()
        binary = _binary()
        if not wrapper or not binary:
            return
        cmd = shlex.join([binary, *paths])
        Gio.Subprocess.new([wrapper, cmd], Gio.SubprocessFlags.NONE)

    def _selected_paths(self, files):
        paths = []
        seen = set()
        for file in files:
            location = file.get_location()
            if not location:
                continue
            path = location.get_path()
            if path and path not in seen:
                seen.add(path)
                paths.append(path)
        return paths

    def _make_item(self, paths):
        label = (
            "Send via Omarchy-Send"
            if len(paths) == 1
            else f"Send {len(paths)} items via Omarchy-Send"
        )
        item = Nautilus.MenuItem(
            name="OmarchySendNautilus::send",
            label=label,
            icon="omarchy-send",
        )
        item.connect("activate", self._on_activate, paths)
        return item

    def _on_activate(self, _menu, paths):
        self._launch(paths)

    def get_file_items(self, *args):
        files = args[0] if len(args) == 1 else args[1]
        if not _wrapper() or not _binary():
            return []
        paths = self._selected_paths(files)
        if not paths:
            return []
        return [self._make_item(paths)]
PY
  fi
  echo "==> Added Nautilus right-click entry 'Send via Omarchy-Send'."
  # Restart Nautilus so the extension loads (windows reopen on demand).
  nautilus -q >/dev/null 2>&1 || true
  fi # nautilus present
else
  echo "==> Headless system — installed as a plain TUI."
fi

# ---- firewall posture ----------------------------------------------------
# Shared by the remote-mode lockdown below and the local-mode public-IP warning.
#
# Tailscale interface: usually tailscale0, but absent when tailscaled runs in
# userspace-networking mode (the default inside containers) — don't hardcode it.
TS_IFACE="$(ip -o link show 2>/dev/null | grep -oE 'tailscale[0-9]+' | head -n1)"

# Container? Under Docker host-networking the port binds the *host's* stack, and
# the firewall belongs on the host, not in this namespace.
IN_CONTAINER=0
if [ -f /.dockerenv ] || grep -qaE 'docker|containerd|kubepods' /proc/1/cgroup 2>/dev/null; then
  IN_CONTAINER=1
fi

# ---- remote server: restrict the port to the Tailscale network -----------
# On a public-IP box, port 53317 would otherwise be reachable from the internet
# (the receiver binds all interfaces). Lock it to the Tailscale interface so it
# only answers over the tailnet. Multicast LAN discovery is link-local and never
# routes off-LAN, so nothing else needs opening. Inside a container the firewall
# can't be applied from here — userspace-networking has no tailscale0 and host-
# networking puts the bind on the host's stack — so we detect that and say so.
if [ "$MODE" = "remote" ]; then
  echo "==> Remote server — restricting port $PORT to the Tailscale network."

  if [ "$IN_CONTAINER" = "1" ] && [ -z "$TS_IFACE" ]; then
    # Container + userspace-networking Tailscale: no tailscale0, and typically no
    # CAP_NET_ADMIN to manage netfilter. A firewall can't be applied from in here.
    echo "    Detected: inside a container with userspace-networking Tailscale"
    echo "    (no tailscale0 interface). The receiver binds all interfaces — and under"
    echo "    Docker host-networking that includes the host's PUBLIC interface."
    echo
    echo "    A firewall CANNOT be applied from in here. Apply it on the HOST:"
    echo "      • if the host already default-denies inbound (e.g. only 22/80/443 open),"
    echo "        $PORT is already blocked from the internet yet still reachable over the"
    echo "        tailnet (tailscaled delivers it via loopback) — nothing more to do."
    echo "      • otherwise, on the host run:  ufw deny $PORT"
    echo "    Strongly recommended in this setup: also set a PIN (--pin <code>)."
  elif [ -n "$TS_IFACE" ] && command -v ufw >/dev/null 2>&1; then
    SUDO=""
    [ "$(id -u)" -ne 0 ] && SUDO="sudo"
    echo "    Tailscale interface: $TS_IFACE"
    echo "    Applying firewall rules (may prompt for sudo):"
    echo "      ${SUDO:+$SUDO }ufw allow in on $TS_IFACE to any port $PORT"
    echo "      ${SUDO:+$SUDO }ufw deny $PORT"
    if $SUDO ufw allow in on "$TS_IFACE" to any port "$PORT" >/dev/null 2>&1 &&
      $SUDO ufw deny "$PORT" >/dev/null 2>&1; then
      echo "    Done — $PORT answers over Tailscale only."
    else
      echo "    Could not apply automatically (need root/sudo). Run the two commands above yourself."
    fi
  else
    if [ -z "$TS_IFACE" ]; then
      echo "    NOTE: no tailscale interface found. If tailscale isn't up yet, install it"
      echo "          and run 'tailscale up', then re-run this installer. If it's running"
      echo "          in userspace-networking mode, firewall the port on the host instead."
      TS_IFACE="tailscale0"
    fi
    echo "    ufw not found. Apply the equivalent in your firewall:"
    echo "      • allow inbound TCP $PORT only on the '$TS_IFACE' interface"
    echo "      • deny inbound $PORT on all other interfaces"
    echo "    nftables example (inet filter, input chain):"
    echo "      iifname \"$TS_IFACE\" tcp dport $PORT accept"
    echo "      tcp dport $PORT drop"
  fi
  echo "    Tip: a PIN adds a second layer — run with --pin <code> (or set it in Settings)."
fi

echo
case ":$PATH:" in
  *":$BIN_DIR:"*) : ;;
  *) echo "Note: $BIN_DIR is not on your PATH. Add it, or run $BIN directly." ;;
esac
echo "Done. Run: omarchy-send"
