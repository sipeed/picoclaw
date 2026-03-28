#!/usr/bin/env python3
"""
PicoClaw + LuCI One-Click Installer for OpenWrt
Supports OpenWrt 24.10 / 25.xx (ARM64 / AMD64 / ARMv7)

Usage:
    pip install paramiko
    python install.py
"""

import paramiko
import sys
import os
import time
import io

sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8', errors='replace')

# NOTE: The LuCI controller and view template are deployed from the actual files
# in this repository (luci/controller/picoclaw.lua and luci/view/picoclaw/main.htm),
# NOT from inline string literals. This ensures the deployed code always matches
# the latest version in the repo.
#
# LuCI controller: luci/controller/picoclaw.lua
# LuCI view template: luci/view/picoclaw/main.htm


def print_step(msg):
    print(f"\n{'='*55}")
    print(f"  {msg}")
    print(f"{'='*55}")


def run_cmd(client, cmd, timeout=120):
    stdin, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    try:
        out = stdout.read().decode('utf-8', errors='replace').strip()
    except:
        out = "(timeout)"
    try:
        err = stderr.read().decode('utf-8', errors='replace').strip()
    except:
        err = ""
    if out:
        print(f"  {out}")
    if err and 'warning' not in err.lower():
        print(f"  [ERR] {err}")
    return out, err


def main():
    print("""
╔══════════════════════════════════════════════════════════╗
║         PicoClaw + LuCI One-Click Installer             ║
║         OpenWrt 24.10 / 25.xx (ARM64/AMD64/ARMv7)      ║
║         https://github.com/sipeed/picoclaw               ║
╚══════════════════════════════════════════════════════════╝
    """)

    host = input("Router IP [default: 192.168.1.1]: ").strip() or "192.168.1.1"
    port = int(input("SSH Port [default: 22]: ").strip() or "22")
    user = input("SSH User [default: root]: ").strip() or "root"
    pwd = input("SSH Password: ").strip()
    if not pwd:
        print("Error: password required")
        sys.exit(1)

    print(f"\nConnecting to {user}@{host}:{port} ...")
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    try:
        client.connect(host, port=port, username=user, password=pwd, timeout=15)
    except Exception as e:
        print(f"Connection failed: {e}")
        sys.exit(1)
    print("Connected!")

    def run(cmd, timeout=120):
        return run_cmd(client, cmd, timeout)

    # Step 1: Detect system
    print_step("Step 1/6: Detect System")
    arch_out, _ = run("uname -m")
    arch = "linux_arm64"
    if "x86" in arch_out:
        arch = "linux_amd64"
    elif "armv7" in arch_out:
        arch = "linux_armv7"
    print(f"  Architecture: {arch_out} -> {arch}")

    release_out, _ = run("cat /etc/openwrt_release 2>/dev/null | grep DISTRIB_DESCRIPTION")
    print(f"  System: {release_out}")

    has_luci, _ = run("test -d /usr/lib/lua/luci && echo 'yes' || echo 'no'")

    # Step 2: Install PicoClaw
    print_step("Step 2/6: Install PicoClaw")
    existing_ver, _ = run("picoclaw version 2>&1 || echo 'not installed'")
    if 'not installed' not in existing_ver and existing_ver:
        print(f"  Already installed: {existing_ver}")
        reinstall = input("  Reinstall? (y/N): ").strip().lower()
        if reinstall != 'y':
            print("  Skipped")
        else:
            run(f"curl -L -o /tmp/picoclaw_new 'https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_{arch}' --max-time 120")
            run("chmod +x /tmp/picoclaw_new; cp /usr/bin/picoclaw /usr/bin/picoclaw.bak 2>/dev/null; mv /tmp/picoclaw_new /usr/bin/picoclaw")
            run("picoclaw version 2>&1")
    else:
        print("  Downloading PicoClaw...")
        run(f"curl -L -o /tmp/picoclaw 'https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_{arch}' --max-time 120")
        run("chmod +x /tmp/picoclaw; mkdir -p /usr/bin; cp /tmp/picoclaw /usr/bin/picoclaw; rm -f /tmp/picoclaw")
        run("picoclaw version 2>&1")

    # Step 3: init.d service
    print_step("Step 3/6: Create init.d Service")
    sftp = client.open_sftp()
    initd_script = open(os.path.join(os.path.dirname(__file__), 'scripts', 'picoclaw.init'), 'r').read()
    with sftp.open('/etc/init.d/picoclaw', 'w') as f:
        f.write(initd_script)
    print("  Written /etc/init.d/picoclaw")
    run("chmod +x /etc/init.d/picoclaw; /etc/init.d/picoclaw enable 2>&1")
    print("  Auto-start enabled")

    # Step 4: Deploy LuCI
    if has_luci.strip() == 'yes':
        print_step("Step 4/6: Deploy LuCI Interface")
        controller_path = os.path.join(os.path.dirname(__file__), 'luci', 'controller', 'picoclaw.lua')
        template_path = os.path.join(os.path.dirname(__file__), 'luci', 'view', 'picoclaw', 'main.htm')

        with sftp.open('/usr/lib/lua/luci/controller/picoclaw.lua', 'w') as f:
            f.write(open(controller_path, 'r').read())
        print("  Controller deployed")

        run("mkdir -p /usr/lib/lua/luci/view/picoclaw")
        with sftp.open('/usr/lib/lua/luci/view/picoclaw/main.htm', 'w') as f:
            f.write(open(template_path, 'r').read())
        print("  Template deployed")

        run("rm -rf /tmp/luci-* 2>/dev/null")
        print("  LuCI cache cleared")
        luci_url = f"http://{host}/cgi-bin/luci/admin/services/picoclaw"
    else:
        print_step("Step 4/6: Skipped (LuCI not detected)")
        luci_url = ""

    sftp.close()

    # Step 5: Init config
    print_step("Step 5/6: Initialize Config")
    run("mkdir -p /root/.picoclaw")
    has_config, _ = run("test -f /root/.picoclaw/config.json && echo 'yes' || echo 'no'")
    if has_config.strip() != 'yes':
        print("  Running picoclaw gateway to generate default config...")
        run("picoclaw gateway >/dev/null 2>&1 &")
        time.sleep(3)
        run("pkill -f 'picoclaw gateway' 2>/dev/null")
        time.sleep(1)
        print("  Default config generated")
    else:
        print("  Config exists, skipped")

    # Step 6: Start service
    print_step("Step 6/6: Start PicoClaw")
    run("pkill -f 'picoclaw gateway' 2>/dev/null")
    time.sleep(1)
    run("/etc/init.d/picoclaw start 2>&1")
    time.sleep(3)

    ps_out, _ = run("ps | grep 'picoclaw gateway' | grep -v grep")
    print(f"\n{'='*55}")
    print(f"  Installation Complete!")
    print(f"{'='*55}")
    if ps_out:
        print(f"  [OK] PicoClaw is running")
    else:
        print(f"  [!] PicoClaw not detected, check: picoclaw gateway")
    if luci_url:
        print(f"\n  LuCI Manager: {luci_url}")
    print(f"  API: http://{host}:18790")
    print(f"  Config: /root/.picoclaw/config.json")

    client.close()
    print("\nDone!")


if __name__ == '__main__':
    try:
        main()
    except KeyboardInterrupt:
        print("\nCancelled")
        sys.exit(0)
    except Exception as e:
        print(f"\nError: {e}")
        sys.exit(1)
