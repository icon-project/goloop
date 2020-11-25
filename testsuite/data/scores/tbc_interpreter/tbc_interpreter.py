from iconservice import *

TAG = 'TBCInterpreter'


class TBCInterpreter(IconScoreBase):
    CALL = 0
    REVERT = 1

    ADDRESS_LEN = 21
    SHORT_LEN = 2

    SCORE_ERROR_BASE = 32

    def __init__(self, db: IconScoreDatabase) -> None:
        super().__init__(db)
        self._name = VarDB('name', db, value_type=str)

    def on_install(self, _name: str) -> None:
        super().on_install()
        self._name.set(_name)

    def on_update(self) -> None:
        super().on_update()

    @eventlog(indexed=1)
    def Event_(self, eventData:int):
        pass

    def Event(self, eventData:str):
        Logger.info(eventData, TAG)

    @external
    def runAndLogResult(self, _code: bytes):
        res = self.run(_code)
        self.Event_(res)

    @external
    def run(self, _code: bytes) -> int:
        self.Event(f'Enter: {self._name.get()}')
        try:
            res = self._runImpl(_code)
            self.Event(f'Exit by Return: {self._name.get()}')
            return res
        except:
            self.Event(f'Exit by Exception: {self._name.get()}')
            raise

    def _runImpl(self, code: bytes) -> int:
        offset = 0
        okCount = 0
        while offset < len(code):
            insn = code[offset]
            offset = offset + 1
            if insn == self.CALL:
                addr = Address.from_bytes(
                    code[offset:offset + self.ADDRESS_LEN]
                )
                offset = offset + self.ADDRESS_LEN
                codeLen = int.from_bytes(
                    code[offset:offset + self.SHORT_LEN],
                    byteorder='big'
                )
                offset = offset + self.SHORT_LEN
                ccode = code[offset: offset + codeLen]
                offset = offset + codeLen
                try:
                    res = self.call(addr, "run", {'_code': ccode})
                    self.Event(res)
                    okCount = okCount + res
                except IconScoreException as e:
                    self.Event(e.message)
                    okCount = okCount + (e.code - self.SCORE_ERROR_BASE)
            elif insn == self.REVERT:
                code = int.from_bytes(
                    code[offset:offset + self.SHORT_LEN],
                    byteorder='big'
                )
                offset = offset + self.SHORT_LEN
                self.Event(f'Exit by Revert: {self._name.get()}')
                revert(code=okCount)
            else:
                self.Event(f'Unexpected insn {insn}')
        return okCount
