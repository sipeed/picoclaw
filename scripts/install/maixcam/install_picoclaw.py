import argparse
import os
import platform
import signal
import shutil
import subprocess
import tarfile
import urllib.request
from pathlib import Path


DEFAULT_ACTION = "install"
START_PICOCLAW = True


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Install or uninstall picoclaw.")
    parser.add_argument(
        "--action",
        choices=["install", "uninstall"],
        default=DEFAULT_ACTION,
        help=f"Action to perform, defaults to '{DEFAULT_ACTION}'.",
    )
    return parser.parse_args()


def detect_arch() -> str:
    machine = platform.machine().lower()

    if machine in {"aarch64", "arm64"}:
        return "arm64"
    if machine in {"riscv64"}:
        return "riscv64"

    return f"unknown ({machine})"


def ensure_picoclaw_dir() -> Path:
    target_dir = Path("/root/picoclaw")
    target_dir.mkdir(parents=True, exist_ok=True)
    return target_dir


def get_download_url(arch: str) -> str:
    if arch == "riscv64":
        return "https://picoclaw-downloads.tos-cn-beijing.volces.com/latest/picoclaw_Linux_riscv64.tar.gz"
    if arch == "arm64":
        return "https://picoclaw-downloads.tos-cn-beijing.volces.com/latest/picoclaw_aarch64.deb"

    raise ValueError(f"Unsupported architecture: {arch}")


def download_package(url: str, target_dir: Path) -> Path:
    file_name = Path(url).name
    target_path = target_dir / file_name
    urllib.request.urlretrieve(url, target_path)
    return target_path


def install_arm64(deb_file: Path) -> None:
    subprocess.run(["dpkg", "-i", str(deb_file)], check=True)


def find_file_recursive(root: Path, file_name: str) -> Path:
    matches = list(root.rglob(file_name))
    if not matches:
        raise FileNotFoundError(f"Required file not found: {file_name}")
    return matches[0]


def install_riscv64(tar_file: Path, work_dir: Path) -> None:
    install_dir = Path("/opt/picoclaw")
    if install_dir.exists():
        shutil.rmtree(install_dir)
    install_dir.mkdir(parents=True, exist_ok=True)

    with tarfile.open(tar_file, "r:gz") as tar:
        tar.extractall(path=install_dir)

    for binary_name in ["picoclaw", "picoclaw-launcher", "picoclaw-launcher-tui"]:
        installed_binary = find_file_recursive(install_dir, binary_name)
        installed_binary.chmod(0o755)

        link_path = Path("/usr/bin") / binary_name
        if link_path.exists() or link_path.is_symlink():
            link_path.unlink()
        link_path.symlink_to(installed_binary)


def cleanup_dir(target_dir: Path) -> None:
    if target_dir.exists():
        shutil.rmtree(target_dir)


def stop_picoclaw() -> None:
    process_names = {"picoclaw", "picoclaw-launcher", "picoclaw-launcher-tui"}
    stopped_any = False
    self_pid = os.getpid()

    try:
        ps_result = subprocess.run(["ps", "-eo", "pid=,args="], capture_output=True, text=True, check=False)
    except FileNotFoundError:
        print("ps command not found, skip stopping picoclaw processes.")
        return

    if ps_result.returncode != 0:
        print("Failed to list processes with ps, skip stopping picoclaw processes.")
        return

    for line in ps_result.stdout.splitlines():
        line = line.strip()
        if not line:
            continue

        parts = line.split(maxsplit=1)
        if not parts:
            continue

        try:
            pid = int(parts[0])
        except ValueError:
            continue

        if pid == self_pid:
            continue

        cmdline = parts[1] if len(parts) > 1 else ""
        executable = os.path.basename(cmdline.split(maxsplit=1)[0]) if cmdline else ""
        if executable not in process_names and not any(f"/{name}" in cmdline for name in process_names):
            continue

        try:
            os.kill(pid, signal.SIGTERM)
            stopped_any = True
        except (ProcessLookupError, PermissionError):
            continue

    if stopped_any:
        print("Stopped picoclaw related processes.")
    else:
        print("No running picoclaw related process found.")


def uninstall_arm64() -> None:
    result = subprocess.run(["dpkg", "-l"], capture_output=True, text=True, check=True)
    package_names = []

    for line in result.stdout.splitlines():
        if line.startswith("ii") and "picoclaw" in line:
            package_names.append(line.split()[1])

    if not package_names:
        print("No installed picoclaw package found for arm64.")
        return

    for package_name in package_names:
        print(f"Removing package: {package_name}")
        subprocess.run(["dpkg", "-r", package_name], check=True)


def uninstall_riscv64() -> None:
    for binary_name in ["picoclaw", "picoclaw-launcher", "picoclaw-launcher-tui"]:
        link_path = Path("/usr/bin") / binary_name
        if link_path.exists() or link_path.is_symlink():
            link_path.unlink()
            print(f"Removed link: {link_path}")

    install_dir = Path("/opt/picoclaw")
    if install_dir.exists():
        shutil.rmtree(install_dir)
        print(f"Removed directory: {install_dir}")


def remove_picoclaw_user_dir() -> None:
    user_dir = Path("/root/.picoclaw")
    if user_dir.exists():
        shutil.rmtree(user_dir)
        print(f"Removed directory: {user_dir}")


def start_picoclaw_launcher() -> None:
    launcher_path = shutil.which("picoclaw-launcher")
    if not launcher_path:
        print("picoclaw-launcher not found in PATH, skip start.")
        return

    env = dict(os.environ)
    env["HOME"] = "/root"
    env.setdefault("NO_COLOR", "1")

    subprocess.Popen(
        [launcher_path, "-no-browser", "-public"],
        env=env,
        stdin=subprocess.DEVNULL,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        start_new_session=True,
        close_fds=True,
    )
    print("Started picoclaw-launcher in background.")


if __name__ == "__main__":
    args = parse_args()
    action = args.action

    arch = detect_arch()
    print(f"Arch: {arch}")
    print(f"Action: {action}")

    if action == "install":
        if arch.startswith("unknown"):
            print("Unsupported arch, exiting.")
            raise SystemExit(1)

        picoclaw_dir = ensure_picoclaw_dir()
        print(f"Ensured directory exists: {picoclaw_dir}")

        download_url = get_download_url(arch)
        print(f"Downloading from: {download_url}")

        downloaded_file = download_package(download_url, picoclaw_dir)
        print(f"Downloaded file: {downloaded_file}")

        if arch == "arm64":
            print("Installing with dpkg...")
            install_arm64(downloaded_file)
        else:
            print("Extracting all files to /opt/picoclaw and creating symlinks...")
            install_riscv64(downloaded_file, picoclaw_dir)

        if START_PICOCLAW:
            start_picoclaw_launcher()

        cleanup_dir(picoclaw_dir)
        print(f"Cleaned up directory: {picoclaw_dir}")
    else:
        stop_picoclaw()

        if arch == "arm64":
            uninstall_arm64()
        elif arch == "riscv64":
            uninstall_riscv64()
        else:
            print("Unknown architecture, skipping uninstallation.")

        remove_picoclaw_user_dir()
