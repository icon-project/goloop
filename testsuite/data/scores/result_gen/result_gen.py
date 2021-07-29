from iconservice import *


class ResultGeneratorInterface(InterfaceScore):
    @interface
    def callRevertWithIndex(self, index: int) -> None:
        pass

    @interface
    def returnStr(self, _value: str) -> str:
        pass


class ResultGenerator(IconScoreBase):
    def __init__(self, db: IconScoreDatabase) -> None:
        super().__init__(db)
        self._bytes_db = DictDB('bytes_db', db, value_type=bytes, depth=2)

    def on_install(self) -> None:
        super().on_install()

    def on_update(self) -> None:
        super().on_update()

    @external
    def callRevertWithIndex(self, index: int) -> None:
        revert(code=index)

    @external(readonly=True)
    def queryRevertWithIndex(self, index: int) -> int:
        revert(code=index)
        return index

    @eventlog
    def RevertCatch(self, code: int) -> None:
        pass

    @external
    def interCallRevertWithIndex(self, addr: Address, index: int) -> None:
        s = self.create_interface_score(addr, ResultGeneratorInterface)
        s.callRevertWithIndex(index)

    @external
    def interCallCatchRevertWithIndex(self, addr: Address, index: int) -> None:
        s = self.create_interface_score(addr, ResultGeneratorInterface)
        try:
            s.callRevertWithIndex(index)
        except IconScoreException as e:
            self.RevertCatch(e.code)

    @external
    def returnStr(self, value: str) -> str:
        return value

    @eventlog
    def ReturnedStr(self, value: str) -> None:
        pass

    @external
    def interCallReturnStr(self, addr: Address, value: str):
        s = self.create_interface_score(addr, ResultGeneratorInterface)
        r_value = s.returnStr(value)
        self.ReturnedStr(r_value)

    @external
    def set_bytes_value(self, addr: Address, index: str):
        data = f'Bytes: {index}'.encode()
        self._bytes_db[addr][index] = data

    @external(readonly=True)
    def get_bytes_value(self, addr: Address, index: str) -> dict:
        result = dict()
        result['index'] = index
        result['address'] = addr
        result['bytes'] = self._bytes_db[addr][index]
        return result

    @external(readonly=True)
    def get_invalid_key(self) -> dict:
        return {
            1: 100
        }
