from iconservice import *

TAG = 'HelloWorld'

class HelloWorld(IconScoreBase):

    def __init__(self, db: IconScoreDatabase) -> None:
        super().__init__(db)

    def on_install(self, name : str) -> None:
        super().on_install()

    def on_update(self, name : str) -> None:
        super().on_update()
    
    @external(readonly=True)
    def name(self) -> str:
        return "HelloWorld"

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
        Logger.info('tokenFallabck is called', TAG)
