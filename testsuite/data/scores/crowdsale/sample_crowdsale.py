from iconservice import *

TAG = 'SampleCrowdsale'


# An interface of token to give a reward to anyone who contributes
class TokenInterface(InterfaceScore):
    @interface
    def transfer(self, _to: Address, _value: int, _data: bytes = None):
        pass


class SampleCrowdsale(IconScoreBase):

    _ADDR_BENEFICIARY = 'addr_beneficiary'
    _ADDR_TOKEN_SCORE = 'addr_token_score'
    _FUNDING_GOAL = 'funding_goal'
    _AMOUNT_RAISED = 'amount_raised'
    _DEAD_LINE = 'dead_line'
    _PRICE = 'price'
    _BALANCES = 'balances'
    _JOINER_LIST = 'joiner_list'
    _FUNDING_GOAL_REACHED = 'funding_goal_reached'
    _CROWDSALE_CLOSED = 'crowdsale_closed'

    ONE_ICX = 10 ** 18

    @eventlog
    def CrowdsaleStarted(self, fundingGoal: int, deadline: int):
        pass

    @eventlog
    def CrowdsaleEnded(self):
        pass

    @eventlog(indexed=3)
    def FundTransfer(self, backer: Address, amount: int, is_contribution: bool):
        pass

    @eventlog(indexed=2)
    def GoalReached(self, recipient: Address, total_amount_raised: int):
        pass

    def __init__(self, db: IconScoreDatabase) -> None:
        super().__init__(db)

        self._addr_beneficiary = VarDB(self._ADDR_BENEFICIARY, db, value_type=Address)
        self._addr_token_score = VarDB(self._ADDR_TOKEN_SCORE, db, value_type=Address)
        self._funding_goal = VarDB(self._FUNDING_GOAL, db, value_type=int)
        self._amount_raised = VarDB(self._AMOUNT_RAISED, db, value_type=int)
        self._dead_line = VarDB(self._DEAD_LINE, db, value_type=int)
        self._price = VarDB(self._PRICE, db, value_type=int)
        self._balances = DictDB(self._BALANCES, db, value_type=int)
        self._joiner_list = ArrayDB(self._JOINER_LIST, db, value_type=Address)
        self._funding_goal_reached = VarDB(self._FUNDING_GOAL_REACHED, db, value_type=bool)
        self._crowdsale_closed = VarDB(self._CROWDSALE_CLOSED, db, value_type=bool)

    def on_install(self, _fundingGoalInIcx: int, _tokenScore: Address, _durationInBlocks: int) -> None:
        """
        Called when this SCORE first deployed.

        :param _fundingGoalInIcx: The funding goal of this crowdsale, in ICX
        :param _tokenScore: SCORE address of token that will be used for the rewards
        :param _durationInBlocks: the sale duration is given in number of blocks
        """
        super().on_install()

        Logger.debug(f'on_install: fundingGoalInIcx={_fundingGoalInIcx}', TAG)
        Logger.debug(f'on_install: tokenScore={_tokenScore}', TAG)
        Logger.debug(f'on_install: durationInBlocks={_durationInBlocks}', TAG)

        if _fundingGoalInIcx < 0:
            revert("Funding goal cannot be less than zero")

        if _durationInBlocks < 0:
            revert("Duration cannot be less than zero")

        # The exchange ratio to ICX is 1:1
        icx_cost_of_each_token = 1

        self._addr_beneficiary.set(self.msg.sender)
        self._addr_token_score.set(_tokenScore)
        self._funding_goal.set(_fundingGoalInIcx * self.ONE_ICX)
        self._dead_line.set(self.block.height + _durationInBlocks)
        price = int(icx_cost_of_each_token)
        self._price.set(price)

        self._funding_goal_reached.set(False)
        self._crowdsale_closed.set(True)  # Crowdsale closed by default

    def on_update(self) -> None:
        super().on_update()

    @external
    def tokenFallback(self, _from: Address, _value: int, _data: bytes):
        """
        Implements `tokenFallback` in order for the SCORE
        to receive initial tokens to reward to the contributors
        """

        # Checks if the caller is a Token SCORE address that this SCORE is interested in.
        if self.msg.sender != self._addr_token_score.get():
            revert("Unknown token address")

        # Depositing tokens can only be done by owner
        if _from != self.owner:
            revert("Invalid sender")

        if _value < 0:
            revert("Depositing value cannot be less than zero")

        # start Crowdsale hereafter
        self._crowdsale_closed.set(False)
        Logger.debug(f'tokenFallback: token supply = "{_value}"', TAG)
        self.CrowdsaleStarted(self._funding_goal.get(), self._dead_line.get())

    @payable
    def fallback(self):
        """
        Called when anyone sends funds to the SCORE.
        This SCORE regards it as a contribution.
        """
        if self._crowdsale_closed.get():
            revert('Crowdsale is closed.')

        # Accepts the contribution
        amount = self.msg.value
        self._balances[self.msg.sender] = self._balances[self.msg.sender] + amount
        self._amount_raised.set(self._amount_raised.get() + amount)
        value = int(amount / self._price.get())
        data = b'called from Crowdsale'

        # Gives tokens to the contributor as a reward
        token_score = self.create_interface_score(self._addr_token_score.get(), TokenInterface)
        token_score.transfer(self.msg.sender, value, data)

        if self.msg.sender not in self._joiner_list:
            self._joiner_list.put(self.msg.sender)

        self.FundTransfer(self.msg.sender, amount, True)
        Logger.debug(f'FundTransfer({self.msg.sender}, {amount}, True)', TAG)

    @external(readonly=True)
    def totalJoinerCount(self) -> int:
        """
        Returns the number of contributors.

        :return: the number of contributors
        """
        return len(self._joiner_list)

    def _after_dead_line(self) -> bool:
        # Checks if it has been reached to the deadline block
        Logger.debug(f'after_dead_line: block.height = {self.block.height}', TAG)
        Logger.debug(f'after_dead_line: dead_line()  = {self._dead_line.get()}', TAG)
        return self.block.height >= self._dead_line.get()

    @external
    def checkGoalReached(self):
        """
        Checks if the goal has been reached and ends the campaign.
        """
        if self._after_dead_line():
            if not self._crowdsale_closed.get():
                self._crowdsale_closed.set(True)
                self.CrowdsaleEnded()

            if self._amount_raised.get() >= self._funding_goal.get():
                self._funding_goal_reached.set(True)
                self.GoalReached(self._addr_beneficiary.get(), self._amount_raised.get())
                Logger.debug(f'Goal reached!', TAG)

    @external
    def safeWithdrawal(self):
        """
        Withdraws the funds.

        If the funding goal has been reached, sends the entire amount to the beneficiary.
        If the goal was not reached, each contributor can withdraw the amount they contributed.
        """
        if self._after_dead_line():
            # each contributor can withdraw the amount they contributed if the goal was not reached
            if not self._funding_goal_reached.get():
                amount = self._balances[self.msg.sender]
                self._balances[self.msg.sender] = 0
                if amount > 0:
                    if self.icx.send(self.msg.sender, amount):
                        self.FundTransfer(self.msg.sender, amount, False)
                        Logger.debug(f'FundTransfer({self.msg.sender}, {amount}, False)', TAG)
                    else:
                        self._balances[self.msg.sender] = amount

            # The sales target has been met. Owner can withdraw the contribution.
            if self._funding_goal_reached.get() and self._addr_beneficiary.get() == self.msg.sender:
                if self.icx.send(self._addr_beneficiary.get(), self._amount_raised.get()):
                    self.FundTransfer(self._addr_beneficiary.get(), self._amount_raised.get(), False)
                    Logger.debug(f'FundTransfer({self._addr_beneficiary.get()},'
                                 f'{self._amount_raised.get()}, False)', TAG)
                    # reset amount_raised
                    self._amount_raised.set(0)
                else:
                    # if the transfer to beneficiary fails, unlock contributors balance
                    Logger.debug(f'Failed to send to beneficiary!', TAG)
                    self._funding_goal_reached.set(False)
