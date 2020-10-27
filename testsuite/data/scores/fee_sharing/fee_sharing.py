from iconservice import *


class FeeSharing(IconScoreBase):

    @eventlog(indexed=1)
    def ValueSet(self, address: Address, proportion: int): pass

    def __init__(self, db: IconScoreDatabase) -> None:
        super().__init__(db)

        self._whitelist = DictDB("whitelist", db, int)
        self._value = VarDB("value", db, str)

    def on_install(self) -> None:
        super().on_install()

        self._value.set("No value")

    def on_update(self) -> None:
        super().on_update()

    def _check_owner(self):
        if self.tx.origin != self.owner:
            revert("Invalid SCORE owner")

    @staticmethod
    def _check_proportion(proportion: int):
        if not (0 <= proportion <= 100):
            revert(f"Invalid proportion: {proportion}")

    @external(readonly=True)
    def getProportion(self, address: Address) -> int:
        return self._whitelist[address]

    @external
    def addToWhitelist(self, address: Address, proportion: int = 100):
        self._check_owner()
        self._check_proportion(proportion)

        self._whitelist[address] = proportion

    @external(readonly=True)
    def getValue(self) -> str:
        return self._value.get()

    @external
    def setValue(self, value: str):
        self._value.set(value)

        proportion: int = self._whitelist[self.tx.origin]
        self.set_fee_sharing_proportion(proportion)

        self.ValueSet(self.tx.origin, proportion)
