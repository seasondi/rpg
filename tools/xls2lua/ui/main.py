import subprocess
import threading


def read_output(process, name):
    for line in iter(process.stdout.readline, b''):
        print(f"{name}: {line.decode().strip()}")


if __name__ == "__main__":
    p = subprocess.Popen("export_table.exe", stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    t = threading.Thread(target=read_output, args=(p, "export_table.exe"))
    t.start()
    t.join()

