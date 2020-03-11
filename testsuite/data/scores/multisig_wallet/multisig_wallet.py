# -*- coding: utf-8 -*-

# Copyright 2018 ICON Foundation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from .type_converter.type_converter import params_type_converter
from .qualification_check.qualification_check import *
from .transaction import Transaction


class MultiSigWallet(IconScoreBase):
    _MAX_WALLET_OWNER_COUNT = 50
    _MAX_DATA_REQUEST_AMOUNT = 50

    @eventlog(indexed=2)
    def Confirmation(self, _sender: Address, _transactionId: int):
        pass

    @eventlog(indexed=2)
    def Revocation(self, _sender: Address, _transactionId: int):
        pass

    @eventlog(indexed=1)
    def Submission(self, _transactionId: int):
        pass

    @eventlog(indexed=1)
    def Execution(self, _transactionId: int):
        pass

    @eventlog(indexed=1)
    def ExecutionFailure(self, _transactionId: int):
        pass

    @eventlog(indexed=1)
    def Deposit(self, _sender: Address, _value: int):
        pass

    @eventlog(indexed=1)
    def DepositToken(self, _sender: Address, _value: int, _data: bytes):
        pass

    @eventlog(indexed=1)
    def WalletOwnerAddition(self, _walletOwner: Address):
        pass

    @eventlog(indexed=1)
    def WalletOwnerRemoval(self, _walletOwner: Address):
        pass

    @eventlog
    def RequirementChange(self, _required: int):
        pass

    def __init__(self, db: IconScoreDatabase) -> None:
        super().__init__(db)
        # store transaction instance as a serialized bytes
        # _transactions's key: transaction id(int type)
        self._transactions = DictDB("transactions", db, value_type=bytes)
        # store wallet owners' confirmations of each transaction
        # _confirmations's key: transaction id(int type), address(Address type)
        self._wallet_owners = ArrayDB("wallet_owners", db, value_type=Address)
        self._confirmations = DictDB("confirmations", db, value_type=bool, depth=2)
        self._required = VarDB("required", db, value_type=int)
        self._transaction_count = VarDB('transactionCount', db, value_type=int)

    def on_install(self, _walletOwners: str, _required: int) -> None:
        super().on_install()

        wallet_owner_list = _walletOwners.replace(" ", "").split(",")
        self._check_requirement(len(wallet_owner_list), _required)

        for wallet_owner in wallet_owner_list:
            wallet_owner_address = Address.from_string(wallet_owner)
            self._wallet_owners.put(wallet_owner_address)

        self._required.set(_required)
        self._transaction_count.set(0)

    def on_update(self) -> None:
        super().on_update()

    @staticmethod
    def _check_params_format_convertible(json_formatted_params: str):
        # when user input None as a _params' value,
        # this will be changed to "" when creating Transaction instance.
        # "" will be changed to {} when finally execute transaction. so doesn't check format
        if json_formatted_params != "" and json_formatted_params is not None:
            try:
                params = json_loads(json_formatted_params)
                for param in params:
                    params_type_converter(param["type"], param["value"])
            except ValueError as e:
                revert(f"json format error: {e}")
            except IconScoreException as e:
                revert(f"{e}")
            except:
                revert("can not convert 'params' json data, check the 'params' parameter")

    @staticmethod
    def _only_positive_number(*args):
        for number in args:
            if number < 0:
                revert("only positive number is accepted")

    def _wallet_owner_does_not_exist(self, wallet_owner: Address):
        if wallet_owner in self._wallet_owners:
            revert(f"{wallet_owner} already exists as an owner of the wallet")

    def _wallet_owner_exist(self, wallet_owner: Address):
        if wallet_owner not in self._wallet_owners:
            revert(f"{wallet_owner} is not an owner of wallet")

    def _transaction_exists(self, transaction_id: int):
        if self._transactions[transaction_id] is None \
                or self._transaction_count.get() <= transaction_id:
            revert(f"transaction id '{transaction_id}' is not exist")

    def _confirmed(self, transaction_id: int, wallet_owner: Address):
        if not self._confirmations[transaction_id][wallet_owner]:
            revert(f"{wallet_owner} has not confirmed to the transaction id '{transaction_id}' yet")

    def _not_confirmed(self, transaction_id: int, wallet_owner: Address):
        if self._confirmations[transaction_id][wallet_owner]:
            revert(f"{wallet_owner} has already confirmed to the transaction '{transaction_id}'")

    def _not_executed(self, transaction_id: int):
        # before call this method, check if transaction is exists(use transaction_exists method)
        if self._transactions[transaction_id][0] == 1:
            revert(f"transaction id '{transaction_id}' has already been executed")

    def _check_requirement(self, wallet_owner_count: int, required: int):
        if wallet_owner_count > self._MAX_WALLET_OWNER_COUNT or \
                required > wallet_owner_count or \
                required <= 0 or \
                wallet_owner_count == 0:
            revert("invalid requirement")

    @payable
    def fallback(self):
        if self.msg.value > 0:
            self.Deposit(self.msg.sender, self.msg.value)

    @external
    def tokenFallback(self, _from: Address, _value: int, _data: bytes):
        if _value > 0:
            self.DepositToken(_from, _value, _data)

    @external
    def submitTransaction(self, _destination: Address,
                          _method: str = "", _params: str = "", _value: int = 0, _description: str = ""):
        self._wallet_owner_exist(self.msg.sender)
        # prevent failure of executing transaction caused by 'params' conversion problems
        self._check_params_format_convertible(_params)
        self._only_positive_number(_value)

        # add transaction
        transaction_id = self._add_transaction(_destination, _method, _params, _value, _description)
        # confirm_transaction
        self.confirmTransaction(transaction_id)

    @external
    def confirmTransaction(self, _transactionId: int):
        self._wallet_owner_exist(self.msg.sender)
        self._transaction_exists(_transactionId)
        self._not_confirmed(_transactionId, self.msg.sender)

        self._confirmations[_transactionId][self.msg.sender] = True

        self.Confirmation(self.msg.sender, _transactionId)

        self._execute_transaction(_transactionId)

    @external
    def revokeTransaction(self, _transactionId: int):
        self._wallet_owner_exist(self.msg.sender)
        self._transaction_exists(_transactionId)
        self._not_executed(_transactionId)
        self._confirmed(_transactionId, self.msg.sender)

        self._confirmations[_transactionId][self.msg.sender] = False

        self.Revocation(self.msg.sender, _transactionId)

    def _add_transaction(self, destination: Address, method: str, params: str, value: int, description: str) -> int:
        transaction = Transaction.create_transaction_with_validation(destination=destination,
                                                                     method=method,
                                                                     params=params,
                                                                     value=value,
                                                                     description=description)
        transaction_id = self._transaction_count.get()

        self._transactions[transaction_id] = transaction.to_bytes()
        self._transaction_count.set(transaction_id + 1)

        self.Submission(transaction_id)
        return transaction_id

    def _execute_transaction(self, transaction_id: int):
        # as this method can't be called from other SCORE or EOA, doesn't check owner, transactions_id, confirmations.
        if self._is_confirmed(transaction_id):
            if self._external_call(self._transactions[transaction_id]):
                self._transactions[transaction_id] = True.to_bytes(1, "big") + self._transactions[transaction_id][1:]

                self.Execution(transaction_id)
            else:
                self.ExecutionFailure(transaction_id)

    def _external_call(self, serialized_tx: bytes) -> bool:
        transaction = Transaction.from_bytes(serialized_tx)

        # if method == "" -> None
        method_name = None if transaction.method == "" else transaction.method
        # if params == "" -> {}
        method_params = {}
        if transaction.params != "":
            params = json_loads(transaction.params)
            for param in params:
                method_params[param["name"]] = params_type_converter(param["type"], param["value"])
        try:
            if transaction.destination.is_contract:
                self.call(addr_to=transaction.destination,
                          func_name=method_name,
                          kw_dict=method_params,
                          amount=transaction.value)
            else:
                self.icx.transfer(transaction.destination, transaction.value)
            execute_result = True
        except:
            execute_result = False

        return execute_result

    def _is_confirmed(self, transaction_id) -> bool:
        count = 0
        for wallet_owner in self._wallet_owners:
            if self._confirmations[transaction_id][wallet_owner]:
                count += 1

        return count == self._required.get()

    @only_wallet
    @external
    def addWalletOwner(self, _walletOwner: Address):
        self._wallet_owner_does_not_exist(_walletOwner)
        # check if owner's count exceed '_MAX_OWNER_COUNT'
        self._check_requirement(len(self._wallet_owners) + 1, self._required.get())

        self._wallet_owners.put(_walletOwner)

        self.WalletOwnerAddition(_walletOwner)

    @only_wallet
    @external
    def replaceWalletOwner(self, _walletOwner: Address, _newWalletOwner: Address):
        self._wallet_owner_exist(_walletOwner)
        self._wallet_owner_does_not_exist(_newWalletOwner)

        for idx, wallet_owner in enumerate(self._wallet_owners):
            if wallet_owner == _walletOwner:
                self._wallet_owners[idx] = _newWalletOwner
                break

        self.WalletOwnerRemoval(_walletOwner)
        self.WalletOwnerAddition(_newWalletOwner)

    @only_wallet
    @external
    def removeWalletOwner(self, _walletOwner: Address):
        self._wallet_owner_exist(_walletOwner)
        # if all owners are removed, this contract can not be executed.
        # so check if _owner is only one left in this wallet
        wallet_owners_count = len(self._wallet_owners)
        self._check_requirement(wallet_owners_count - 1, self._required.get())

        for idx, owner in enumerate(self._wallet_owners):
            if owner == _walletOwner:
                if idx == wallet_owners_count - 1:
                    self._wallet_owners.pop()
                else:
                    self._wallet_owners[idx] = self._wallet_owners.pop()
                break

        self.WalletOwnerRemoval(_walletOwner)

    @only_wallet
    @external
    def changeRequirement(self, _required: int):
        self._check_requirement(len(self._wallet_owners), _required)

        self._required.set(_required)

        self.RequirementChange(_required)

    @external(readonly=True)
    def getRequirement(self) -> int:
        return self._required.get()

    @external(readonly=True)
    def getTransactionInfo(self, _transactionId: int) -> dict:
        if self._transactions[_transactionId] is not None:
            transaction = Transaction.from_bytes(self._transactions[_transactionId])
            tx_dict = transaction.to_dict()
            tx_dict["_transactionId"] = _transactionId
            return tx_dict
        else:
            return {}

    @external(readonly=True)
    def getTransactionsExecuted(self, _transactionId: int) -> bool:
        if self._transactions[_transactionId] is not None:
            return bool(self._transactions[_transactionId][0])
        else:
            return False

    @external(readonly=True)
    def checkIfWalletOwner(self, _walletOwner: Address) -> bool:
        return _walletOwner in self._wallet_owners

    @external(readonly=True)
    def getWalletOwnerCount(self) -> int:
        return len(self._wallet_owners)

    @external(readonly=True)
    def getWalletOwners(self, _offset: int, _count: int) -> list:
        self._only_positive_number(_offset, _count)

        wallet_owner_list = []
        wallet_owners_count = len(self._wallet_owners)

        for idx in range(_offset, _offset + _count):
            if idx >= wallet_owners_count:
                break
            wallet_owner_list.append(str(self._wallet_owners[idx]))

        return wallet_owner_list

    @external(readonly=True)
    def getConfirmationCount(self, _transactionId: int) -> int:
        count = 0
        for wallet_owner in self._wallet_owners:
            if self._confirmations[_transactionId][wallet_owner]:
                count += 1
        return count

    @external(readonly=True)
    def getConfirmations(self, _offset: int, _count: int, _transactionId: int) -> list:
        self._only_positive_number(_offset, _count)

        confirmed_wallet_owners = []
        wallet_owners_count = len(self._wallet_owners)

        for idx in range(_offset, _offset + _count):
            if idx >= wallet_owners_count:
                break
            if self._confirmations[_transactionId][self._wallet_owners[idx]]:
                confirmed_wallet_owners.append(str(self._wallet_owners[idx]))

        return confirmed_wallet_owners

    @external(readonly=True)
    def getTransactionCount(self, _pending: bool = True, _executed: bool = True) -> int:
        tx_count = 0
        for tx_id in range(self._transaction_count.get()):
            if (_pending and not self._transactions[tx_id][0]) or (_executed and self._transactions[tx_id][0]):
                tx_count += 1

        return tx_count

    @external(readonly=True)
    def getTransactionList(self, _offset: int, _count: int, _pending: bool = True, _executed: bool = True) -> list:
        self._only_positive_number(_offset, _count)

        if _count > self._MAX_DATA_REQUEST_AMOUNT:
            revert("requests that exceed the allowed amount")

        transaction_list = []
        total_transaction_count = self._transaction_count.get()

        # prevent searching not existed transaction
        _count = _offset + _count if total_transaction_count >= _offset + _count else total_transaction_count

        for tx_id in range(_offset, _count):
            if (_pending and not self._transactions[tx_id][0]) or (_executed and self._transactions[tx_id][0]):
                transaction = Transaction.from_bytes(self._transactions[tx_id])

                tx_dict = transaction.to_dict()
                tx_dict["_transactionId"] = tx_id
                transaction_list.append(tx_dict)

        return transaction_list
