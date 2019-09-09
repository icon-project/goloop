from iconservice import *

TAG = 'Governance'


# An interface of token to give a reward to anyone who contributes
class SystemInterface(InterfaceScore):
    @interface
    def setRevision(self, code: int):
        pass
    @interface
    def acceptScore(self, txHash: bytes):
        pass
    @interface
    def rejectScore(self, txHash: bytes):
        pass
    @interface
    def blockScore(self, address: Address):
        pass
    @interface
    def unblockScore(self, address: Address):
        pass
    @interface
    def setStepPrice(self, price: int):
        pass
    @interface
    def setStepCost(self, type: str, cost: int):
        pass
    @interface
    def setMaxStepLimit(self, contextType: str, limit: int):
        pass
    @interface
    def addDeployer(self, address: Address):
        pass
    @interface
    def removeDeployer(self, address: Address):
        pass
    @interface
    def grantValidator(self, address: Address):
        pass
    @interface
    def revokeValidator(self, address: Address):
        pass
    @interface
    def addMember(self, address: Address):
        pass
    @interface
    def removeMember(self, address: Address):
        pass
    @interface
    def addLicense(self, contentId: str):
        pass
    @interface
    def removeLicense(self, contentId: str):
        pass
    @interface
    def setTimestampThreshold(self, threshold: int):
        pass

class Governance(IconScoreBase):

    def __init__(self, db: IconScoreDatabase) -> None:
        super().__init__(db)
        self.system_score = self.create_interface_score(Address.from_string("cx0000000000000000000000000000000000000000"), SystemInterface)

    def on_install(self, name : str, value : int) -> None:
        print("install governance : ", name, ", value : ", value)
        """
        Called when this SCORE first deployed.

        :param _fundingGoalInIcx: The funding goal of this crowdsale, in ICX
        :param _tokenScore: SCORE address of token that will be used for the rewards
        :param _durationInBlocks: the sale duration is given in number of blocks
        """
        super().on_install()


    def on_update(self) -> None:
        super().on_update()

    @payable
    def fallback(self):
        """
        Called when anyone sends funds to the SCORE.
        This SCORE regards it as a contribution.
        """
        Logger.debug(f'FundTransfer({self.msg.sender}, {self.msg.value}, True)', TAG)

    @external
    def setRevision(self, code: int):
        self.system_score.setRevision(code)

    @external
    def acceptScore(self, txHash: bytes):
        self.system_score.acceptScore(txHash)

    @external
    def rejectScore(self, txHash: bytes):
        self.system_score.rejectScore(txHash)

    @external
    def blockScore(self, address: Address):
        self.system_score.blockScore(address)

    @external
    def unblockScore(self, address: Address):
        self.system_score.unblockScore(address)

    @external
    def setStepPrice(self, price: int):
        self.system_score.setStepPrice(price)

    @external
    def setStepCost(self, type: str, cost: int):
        self.system_score.setStepCost(type, cost)

    @external
    def setMaxStepLimit(self, contextType: str, limit: int):
        self.system_score.setMaxStepLimit(contextType, limit)

    @external
    def grantValidator(self, address: Address):
        self.system_score.grantValidator(address)
    @external
    def revokeValidator(self, address: Address):
        self.system_score.revokeValidator(address)
    @external
    def addMember(self, address: Address):
        self.system_score.addMember(address)

    @external
    def removeMember(self, address: Address):
        self.system_score.removeMember(address)
    @external
    def addDeployer(self, address: Address):
        self.system_score.addDeployer(address)

    @external
    def removeDeployer(self, address: Address):
        self.system_score.removeDeployer(address)

    @external
    def addLicense(self, contentId: str):
        self.system_score.addLicense(contentId)

    @external
    def removeLicense(self, contentId: str):
        self.system_score.removeLicense(contentId)

    @external
    def setTimestampThreshold(self, threshold: int):
        self.system_score.setTimestampThreshold(threshold)

    @external(readonly=True)
    def updated(self) -> bool:
        return True
