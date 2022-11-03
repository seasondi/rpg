import _thread
import select
import telnetlib
import time


def help():
    print("""控制台调试工具
    help: 帮助
    exit/quit: 退出
    以\\结尾: 多行输入
    """)


def pre_process_cmd(data):
    return (data + "\n").encode("ascii")


def on_received_console_data(data):
    data = data.decode("ascii").replace("\r\n", "")
    data = str(data).rstrip("\n")
    print(str(data))


class TelnetConsole:
    def __init__(self, host, port):
        self.quit = False
        self.host = host
        self.port = port
        self.consoleInstance = None

    def close(self):
        if self.quit:
            return
        self.quit = True
        if self.consoleInstance:
            self.consoleInstance.close()
        self.consoleInstance = None
        self.host = ""
        self.port = 0
        print("bye bye")

    def run(self):
        try:
            self.consoleInstance = telnetlib.Telnet(self.host, self.port)
        except Exception:
            print("服务器连接失败\n")
            self.close()
            return

        _thread.start_new_thread(self.receive_console_data, ())

        while True:
            data = input("> ")
            if data.endswith("\\"):
                data = data[::-1].replace("\\", " ")[::-1]
                all_data = data
                while True:
                    data = input(">>> ")
                    all_data = all_data + data
                    if not data.endswith("\\"):
                        break
                    else:
                        all_data = all_data[::-1].replace("\\", " ")[::-1]
                data = all_data

            if self.quit or data == "exit" or data == "quit":
                break
            else:
                if data == "help":
                    help()
                else:
                    self.consoleInstance.write(pre_process_cmd(data))
                    time.sleep(0.1)
                    # data = self.consoleInstance.read_until(b"\r\n", 5)
                    # if data is None:
                    #     break
                    # on_received_console_data(data)

        self.close()

    def receive_console_data(self):
        fd = self.consoleInstance.fileno()
        while True:
            if self.quit:
                return
            readable, _, __ = select.select([fd, ], [], [], 0.1)
            if fd in readable:
                try:
                    data = self.consoleInstance.read_very_eager()
                    on_received_console_data(data)
                except Exception as ex:
                    print(ex)
                    self.close()
                    return


if __name__ == "__main__":
    c = TelnetConsole("127.0.0.1", 7000)
    c.run()
