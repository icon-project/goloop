from iconservice import *

TAG = 'TBCInterpreter'


class TBCInterpreter(IconScoreBase):
    CALL = 0
    REVERT = 1

    ADDRESS_LEN = 21
    SHORT_LEN = 2

    def __init__(self, db: IconScoreDatabase) -> None:
        super().__init__(db)
        self._name = VarDB('name', db, value_type=str)
        self._res = ''

    def on_install(self, _name: str) -> None:
        super().on_install()
        self._name.set(_name)

    def on_update(self) -> None:
        super().on_update()

    @eventlog(indexed=1)
    def Event_(self, eventData:str):
        pass

    def Event(self, eventData:str):
        if len(self._res) > 0:
            self._res = self._res + '\n'
        self._res = self._res + eventData

    @external
    def runAndLogResult(self, _code: bytes):
        res = self.run(_code)
        self.Event_(res)

    @external
    def run(self, _code: bytes) -> str:
        self._res = ''
        self.Event(f'Enter: {self._name.get()}')
        try:
            self._runImpl(_code)
            self.Event(f'Exit by Return: {self._name.get()}')
        except:
            self.Event(f'Exit by Exception: {self._name.get()}')
            raise
        return self._res

    def _runImpl(self, code: bytes):
        offset = 0
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
                except IconScoreException as e:
                    self.Event(e.message)
                    pass
            elif insn == self.REVERT:
                code = int.from_bytes(
                    code[offset:offset + self.SHORT_LEN],
                    byteorder='big'
                )
                offset = offset + self.SHORT_LEN
                self.Event(f'Exit by Revert: {self._name.get()}')
                revert(self._res, code)
            else:
                self.Event(f'Unexpected insn {insn}')
