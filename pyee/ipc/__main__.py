from .client import *


def main():
    client = Client()
    client.connect("/tmp/test")
    msg, req_id, data = client.send_and_receive(1, "From Python")
    print(msg, req_id, data)


if __name__ == "__main__":
    main()
