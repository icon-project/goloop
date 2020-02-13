from iconservice import *

TAG = 'EventGen'


class EventGen(IconScoreBase):
    def __init__(self, db: IconScoreDatabase) -> None:
        super().__init__(db)

    def on_install(self, name: str) -> None:
        super().on_install()
        Logger.info(f'on_install: name={name}', TAG)

    def on_update(self) -> None:
        super().on_update()

    @eventlog(indexed=3)
    def Event(self, _addr: Address, _int: int, _bytes: bytes):
        pass

    @external
    def generate(self, _addr: Address, _int: int, _bytes: bytes):
        self.Event(_addr, _int, _bytes)

    @eventlog(indexed=3)
    def EventEx(self, _bool: bool, _int: int, _str: str, _addr: Address, _bytes: bytes):
        pass

    @external
    def generateNullByIndex(self, _idx: int):
        args = [True, 1, "test", Address.from_string("hx0000000000000000000000000000000000000000"), bytes([1])]
        args[_idx] = None
        self.EventEx(*args)

    @payable
    def fallback(self):
        Logger.info('fallback is called', TAG)

