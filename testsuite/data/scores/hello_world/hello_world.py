from iconservice import *

TAG = 'HelloWorld'
GOV_SCORE_ADDRESS = Address.from_prefix_and_int(AddressPrefix.CONTRACT, 1)


class InterCallInterface(InterfaceScore):
    @interface
    def infinite_intercall(self, _to: Address, call_cnt: int):
        pass

    @interface
    def doNothing(self):
        pass

    @interface
    def doRevert(self):
        pass

    @interface
    def payableDoNothing(self):
        pass

    @interface
    def payableDoRevert(self):
        pass

    @interface
    def readOnlyDoNothing(self) -> str:
        pass

    @interface
    def readOnlyDoRevert(self) -> str:
        pass


class ChainSCORE(InterfaceScore):
    @interface
    def getRevision(self) -> int:
        pass


class Governance(InterfaceScore):
    @interface
    def setRevision(self, code: int):
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
    def balanceOf(self, _owner: Address) -> int:
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

    @external
    def checkRevision(self, code: int):
        score = self.create_interface_score(ZERO_SCORE_ADDRESS, ChainSCORE)
        rev = score.getRevision()
        Logger.debug(f"Revision({rev})")

    @external
    def setRevision(self, code: int):
        score = self.create_interface_score(GOV_SCORE_ADDRESS, Governance)
        score.setRevision(code)

    @external
    def testMaxBufferSize(self, size: int):
        _prev_name = self._name.get()
        # begin test
        _list = []
        for i in range(1024*size):
            _list.append(f'index{i}')
        self._name.set(json_dumps(_list))
        _value = self._name.get()
        Logger.info(f"value len={len(_value)}", TAG)
        # end test
        self._name.set(_prev_name)

    @external
    def callMethodOf(self, to: Address, method: int):
        score = self.create_interface_score(to, InterCallInterface)
        if method == 0:
            score.doNothing()
        elif method == 1:
            score.doRevert()
        elif method == 2:
            score.icx(10).doNothing()
        elif method == 3:
            score.icx(10).doRevert()
        elif method == 4:
            score.icx(10).payableDoNothing()
        elif method == 5:
            score.icx(10).payableDoRevert()
        elif method == 6:
            score.readOnlyDoNothing()
        elif method == 7:
            score.readOnlyDoRevert()
        elif method == 8:
            score.icx(10).readOnlyDoNothing()
        elif method == 9:
            score.icx(10).readOnlyDoRevert()
        elif method == 10:
            self.icx.transfer(to, 10)
        else:
            revert('InvalidMethod')
