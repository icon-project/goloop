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
