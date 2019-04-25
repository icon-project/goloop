from iconservice import *

TAG = 'EventGen'

class EventGen(IconScoreBase):
    def __init__(self, db: IconScoreDatabase) -> None:
        super().__init__(db)

    def on_install(self, name : str) -> None:
        super().on_install()

    def on_update(self) -> None:
        super().on_update()

    @eventlog(indexed=3)
    def Event(self, _addr: Address, _int: int, _bytes: bytes):
        pass

    @external
    def generate(self, _addr: Address, _int: int, _bytes: bytes):
        Logger.info('generate', TAG)
        if _bytes is None:
            _bytes = b'None'
        self.Event(_addr, _int, _bytes)
        return 'OK'

    @payable
    def fallback(self):
        Logger.info('fallback is called', TAG)

