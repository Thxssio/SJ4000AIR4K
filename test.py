import socket 
import sys
import threading
import time
from typing import List, Tuple, Union

class PHeader:
    def __init__(self, magic: Tuple[int, int], length: Tuple[int, int], msg: Tuple[int, int], other: Union[None, List[int]] = None):
        self.magic = magic
        self.length = length
        self.msg = msg
        self.other = other

    def string(self):
        has_ext = self.other is not None
        return f"<PHeader MAGIC={self.magic} LEN={self.length} MSG={self.msg} HAS_EXT=({has_ext})>"


def bytes_to_header(input: bytes) -> Tuple[PHeader, bool]:
    temp = bytearray(8)
    if len(input) >= 8:
        temp[:8] = input[:8]
        magic = tuple(input[:2])
        length = tuple(input[2:4])
        idk = tuple(input[4:6])
        msg = tuple(input[6:8])
        other = list(input[8:]) if len(input) > 8 else None

        return PHeader(magic, length, msg, other), True
    else:
        return None, False


def login_payload() -> bytes:
    username_bytes = b"admin"
    password_bytes = b"12345"
    header = bytearray([171, 205, 0, 128, 0, 0, 1, 16])
    payload = bytearray([0] * 128)
    payload[:len(username_bytes)] = username_bytes
    payload[64:64 + len(password_bytes)] = password_bytes
    header.extend(payload)

    return bytes(header)


def handle_rtp_client(stream: socket):
    t = time.time() + 3
    while time.time() < t:
        payload = stream.recv(1024).decode()

        reqs: List[str] = [s.strip() for s in payload.split('\r\n') if s.strip()]

        if len(reqs) == 3:

            req_head = reqs[0].split()
            req_cseq = reqs[1].split(':')
            req_ua = reqs[2].split(':')
            print(f"=>> {req_head}\n{req_cseq}\n{req_ua} <<=")
            
            if req_head[0] == "OPTIONS":
                print("got options")
                response = "RTSP/1.0 200 OK\r\nCSeq: 2\r\nPublic: DESCRIBE, SETUP, PLAY, PAUSE, RECORD\r\n\r\n"
                stream.send(response.encode())

def get_rtsp_status(code: int, status_msg: str) -> bytes:
    packet = f"RTSP/1.0 {code} {status_msg}\r\n"
    return packet.encode()

def start_rtsp(stream):
    command = bytearray([171, 205, 0, 8, 0, 0, 1, 255])
    payload = bytearray([0] * 8)
    command.extend(payload)
    w = stream.write(command)
    print(f"rtsp => wrote {w} bytes")
    if w == 16:
        return True
    return False


def keep_alive(stream: socket):
    logged_in = False
    alive_res = bytearray([171, 205, 0, 0, 0, 0, 1, 19])
    login_acpt = bytearray([171, 205, 0, 129, 0, 0, 1, 17])
    login_res = bytearray(8)
    sleep_time = 1

    while True:
        if not logged_in:
            print("should print 1 time")
            _br = stream.recv_into(login_res)

            if login_res == login_acpt:
                logged_in = True

                s2 = stream.dup()
                _y = start_rtsp(s2)
                print(f"Login Aceitado\n{bytes_to_header(login_res).string()}")
                if _y:
                    thread = threading.Thread(target=rtsp_thread)
                    thread.start()

        _bw = stream.send(alive_res)
        if True:
            pass

        print("FROM ALIVE")
        time.sleep(sleep_time)

def main():
    target = ("192.168.100.1", 6666)

    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as con:
        con.connect(target)
        lbw = con.sendall(login_payload)
        print(f"Login Sent! {lbw}")
        t = threading.Thread(target=keep_alive, args=(con,))
        t.start()
        t.join()

if __name__ == "__main__":
    login_payload = login_payload()
    main()

