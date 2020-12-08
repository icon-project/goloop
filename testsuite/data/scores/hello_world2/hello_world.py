from iconservice import *

TAG = 'HelloWorld2'


class HelloWorld(IconScoreBase):

    def __init__(self, db: IconScoreDatabase) -> None:
        super().__init__(db)
        self._name = VarDB('name', db, value_type=str)

    def on_install(self, name: str) -> None:
        super().on_install()

    def on_update(self, name: str) -> None:
        super().on_update()
        self._name.set(name)
        Logger.info(f"on_update: name={name}", TAG)

    @external(readonly=True)
    def name(self) -> str:
        return self._name.get()

    @external
    def hello(self):
        Logger.info('Hello, world!', TAG)

    @external
    def helloWithName2(self, name: str):
        Logger.info('Hello 2 %s' % name,  TAG)

    @payable
    def fallback(self):
        Logger.info('fallback is called', TAG)

    @external
    def tokenFallback(self, _from: Address, _value: int, _data: bytes):
        Logger.info('tokenFallback is called', TAG)

    @external
    def doNothing(self):
        Logger.info('doNothing', TAG)

    @external
    def doRevert(self):
        Logger.info('doRevert', TAG)
        revert('Abort')

    @payable
    @external
    def payableDoNothing(self):
        Logger.info('payableDoNothing', TAG)

    @payable
    @external
    def payableDoRevert(self):
        Logger.info('payableDoRevert', TAG)
        revert('Abort')

    @external(readonly=True)
    def readOnlyDoNothing(self) -> str:
        Logger.info('readOnlyDoNothing', TAG)
        return ""

    @external(readonly=True)
    def readOnlyDoRevert(self) -> str:
        Logger.info('readOnlyDoRevert', TAG)
        revert('Abort')
        return ""
