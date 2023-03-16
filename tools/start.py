import subprocess
import threading


def read_output(process, name):
    for line in iter(process.stdout.readline, b''):
        print(f"{name}: {line.decode().strip()}")


servers = [("db", 1), ("game", 2), ("gate", 1)]

if __name__ == "__main__":
    all_thread = []

    for info in servers:
        for i in range(0, info[1]):
            target = info[0] + ".exe"
            cmd = "./bin/" + target + " --config=./config/config.json --tag=" + str(i+1)
            p = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            t = threading.Thread(target=read_output, args=(p, target))
            t.start()
            all_thread.append(t)

    for t in all_thread:
        t.join()

