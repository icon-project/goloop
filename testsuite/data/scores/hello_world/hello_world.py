from iconservice import *

TAG = 'HelloWorld'


class InterCallInterface(InterfaceScore):
    @interface
    def infinite_intercall(self, _to: Address, call_cnt: int):
        pass


class HelloWorld(IconScoreBase):
    _NAME = 'name'
    _BALANCES = 'balances'

    def __init__(self, db: IconScoreDatabase) -> None:
        super().__init__(db)
        self._name = VarDB(self._NAME, db, value_type=str)
        self._balances = DictDB(self._BALANCES, db, value_type=int)

    def on_install(self, name: str) -> None:
        super().on_install()
        self._name.set(name)
        Logger.info(f"on_install: name={name}", TAG)

    def on_update(self) -> None:
        super().on_update()

    @external(readonly=True)
    def name(self) -> str:
        return self._name.get()

    @external
    def hello(self):
        Logger.info('Hello, world!', TAG)

    @external
    def helloWithName(self, name: str, age: str = None):
        Logger.info('Hello %s' % name,  TAG)

    @payable
    def fallback(self):
        Logger.info('fallback is called', TAG)

    @external
    def infiniteLoop(self):
        loop_cnt = 1
        while True:
            loop_cnt = loop_cnt + 1

    @external
    @payable
    def transfer(self) -> None:
        Logger.info('Transfer!!', TAG)
        self._balances[self.msg.sender] = self.msg.value

    @external(readonly=True)
    def balanceOf(self, _owner: Address) -> str:
        Logger.info(f"balanceOf : {self._balances[_owner]}", TAG)
        return self._balances[_owner]

    @external
    def infinite_intercall(self, _to: Address, call_cnt: int):
        score = self.create_interface_score(_to, InterCallInterface)
        Logger.debug(f"intercall ({_to}) call_cnt({call_cnt})", TAG)
        score.infinite_intercall(self.address, call_cnt + 1)

    @external
    def transferICX(self, to: Address, amount: int):
        if amount > 0:
            self.icx.transfer(to, amount)
