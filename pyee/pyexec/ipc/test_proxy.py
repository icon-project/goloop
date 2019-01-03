from unittest import TestCase
from .proxy import *


class TestExecutionHandler(TestCase):
    def test_int_to_bytes(self):
        cases = [
            (0, []),
            (0x80, [0x00, 0x80]),
            (-0x80, [0x80]),
            (-255, [0xff, 0x01]),
            (0x1234, [0x12, 0x34]),
        ]
        for c, b in cases:
            b2 = int_to_bytes(c)
            if b2 != bytes(b):
                self.fail(f"Expect {b} but {b2} returned for {c}")

    def test_bytes_to_int(self):
        cases = [
            0,
            0x7f,
            0x80,
            -0x80,
            -0xff,
            0x7fff,
            0x7fffffff,
        ]
        for c in cases:
            bs = int_to_bytes(c)
            c_out = bytes_to_int(bs)
            if c_out != c:
                self.fail()
