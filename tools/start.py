import subprocess
import threading
import time
import signal

games, gates, dbs, admins = [], [], [], []

def read_output(process, name):
    for line in iter(process.stdout.readline, b''):
        print(f"{name}: {line.decode().strip()}")


def stop_processes():
    for p in reversed(gates):
        p.terminate()
        p.wait()

    for p in reversed(games):
        p.terminate()
        p.wait()

    for p in reversed(dbs):
        p.terminate()
        p.wait()

    for p in reversed(admins):
        p.terminate()
        p.wait()


def signal_handler(sig, frame):
    stop_processes()
    exit(0)


servers = [("db", 1), ("game", 3), ("gate", 2), ("admin", 1)]

if __name__ == "__main__":
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)

    all_thread = []

    for info in servers:
        for i in range(0, info[1]):
            target = info[0] + ".exe"
            cmd = "./bin/" + target + " --config=./config/config.json --tag=" + str(i+1)
            p = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            if info[0] == "game":
                games.append(p)
            elif info[0] == "gate":
                gates.append(p)
            elif info[0] == "db":
                dbs.append(p)
            else:
                admins.append(p)

            t = threading.Thread(target=read_output, args=(p, target))
            t.start()
            all_thread.append(t)

    for t in all_thread:
        t.join()

