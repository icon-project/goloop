#!/usr/bin/env python

from iconservice import *

TAG = 'RevertTest'


class StepCounterInterface(InterfaceScore):
    @interface
    def setStep(self, step: int) -> None:
        pass

    @interface
    def increaseStepWith(self, addr: Address, count: int) -> None:
        pass


class StepCounter(IconScoreBase):
    @eventlog(indexed=1)
    def OnStep(self, step: int):
        pass

    def __init__(self, db: IconScoreDatabase) -> None:
        super().__init__(db)
        self._progress = VarDB("progress", db, value_type=int)

    def on_install(self) -> None:
        super().on_install()
        self.OnStep(1)
        self._progress.set(1)

    def on_update(self) -> None:
        super().on_update()

    @external(readonly=True)
    def getStep(self) -> int:
        return self._progress.get()

    @external
    def setStep(self, step: int) -> None:
        v = self._progress.get()
        self.OnStep(step)
        self._progress.set(step)
        if v+1 != step:
            revert(f"Unexpected value {v}+1 != {step}")

    @external
    def resetStep(self, step: int) -> None:
        self.OnStep(step)
        self._progress.set(step)

    @external
    def increaseStep(self) -> None:
        v = self._progress.get()
        self.OnStep(v + 1)
        self._progress.set(v+1)

    @eventlog(indexed=2)
    def ExternalProgress(self, addr: Address, step: int) -> None:
        pass

    @external
    def setStepOf(self, addr: Address, step: int) -> None:
        s = self.create_interface_score(addr, StepCounterInterface)
        self.ExternalProgress(addr, step)
        s.setStep(step)

    @external
    def trySetStepWith(self, addr: Address, step: int) -> None:
        s = self.create_interface_score(addr, StepCounterInterface)
        self.ExternalProgress(addr, step)
        try:
            s.setStep(step)
        except IconScoreException:
            pass
        self.setStep(step)

    @external
    def increaseStepWith(self, addr: Address, count: int) -> None:
        self.increaseStep()
        count = count-1
        if count > 0:
            s = self.create_interface_score(addr, StepCounterInterface)
            s.increaseStepWith(self.address, count)
