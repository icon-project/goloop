
from iconservice import *


class ResultGeneratorInterface(InterfaceScore):
    @interface
    def callRevertWithIndex(self, index: int) -> None:
        pass


class ResultGenerator(IconScoreBase):
    def __init__(self, db: IconScoreDatabase) -> None:
        super().__init__(db)

    def on_install(self) -> None:
        super().on_install()

    def on_update(self) -> None:
        super().on_update()

    @external
    def callRevertWithIndex(self, index: int) -> None:
        self.revert(code=index)

    @external(readonly=True)
    def queryRevertWithIndex(self, index: int) -> int:
        self.revert(code=index)
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
