from unittest import TestCase

from .proxy import *


class TestExecutionHandler(TestCase):
    def test_int_to_bytes(self):
        cases = [
            (0, [0]),
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

    def test_log_level_conversion(self):
        cases = [
            Log.PANIC,
            Log.FATAL,
            Log.WARN,
            Log.INFO,
            Log.DEBUG,
            Log.TRACE,
        ]

        for c in cases:
            s = Log.to_string(c)
            if type(s) is not str:
                self.fail(f"Type of returned {s} isn't string")
            lv = Log.from_string(s)
            if c != lv:
                self.fail(f"Level={c} to {s}, and to {lv}")

    def test_log_invalid_to_string(self):
        cases = [
            Log.PANIC-1,
            Log.TRACE+1,
            "1",
            None
        ]
        for c in cases:
            try:
                s = Log.to_string(c)
                self.fail(f"to_string({c}) returns {s} (exception expected)")
            except Exception:
                pass

    def test_log_invalid_from_string(self):
        cases = [
            "panic2",
            "test",
            "1",
            None
        ]
        for c in cases:
            try:
                s = Log.from_string(c)
                self.fail(f"from_string({c}) returns {s} (exception expected)")
            except Exception:
                pass

