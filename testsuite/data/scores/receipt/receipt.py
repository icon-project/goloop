from iconservice import *

TAG = 'Receipt'


class InterCallInterface(InterfaceScore):
    @interface
    def call_event_log(self, p_log_index: int, p_bool: bool, p_addr: Address, p_int: int, p_bytes: bytes, p_str: str):
        pass

class Receipt(IconScoreBase):
    @eventlog
    def event_log_no_index(self, p_bool: bool, p_addr: Address, p_int: int, p_bytes: bytes, p_str: str):
        pass

    @eventlog(indexed=1)
    def event_log_1_index(self, p_bool: bool, p_addr: Address, p_int: int, p_bytes: bytes, p_str: str):
        pass

    @eventlog(indexed=2)
    def event_log_2_index(self, p_bool: bool, p_addr: Address, p_int: int, p_bytes: bytes, p_str: str):
        pass

    @eventlog(indexed=3)
    def event_log_3_index(self, p_bool: bool, p_addr: Address, p_int: int, p_bytes: bytes, p_str: str):
        pass

    def __init__(self, db: IconScoreDatabase) -> None:
        super().__init__(db)

    def on_install(self) -> None:
        super().on_install()

    def on_update(self) -> None:
        super().on_update()

    @external
    def call_event_log(self, p_log_index: int, p_bool: bool, p_addr: Address, p_int: int, p_bytes: bytes, p_str: str):
        if p_log_index == 0:
            self.event_log_no_index(p_bool, p_addr, p_int, p_bytes, p_str)
        elif p_log_index == 1:
            self.event_log_1_index(p_bool, p_addr, p_int, p_bytes, p_str)
        elif p_log_index == 2:
            self.event_log_2_index(p_bool, p_addr, p_int, p_bytes, p_str)
        elif p_log_index == 3:
            self.event_log_3_index(p_bool, p_addr, p_int, p_bytes, p_str)
        else:
            raise Exception(f'Illegal argument for index {p_log_index})')

    @external
    def inter_call_event_log(self, _to: Address, p_log_index: int, p_bool: bool, p_addr: Address, p_int: int, p_bytes: bytes, p_str: str):
        recipient_score = self.create_interface_score(_to, InterCallInterface)
        recipient_score.call_event_log(p_log_index, p_bool, p_addr, p_int, p_bytes, p_str)

    @external
    def event_log_and_inter_call(self, _to: Address, p_log_index: int, p_bool: bool, p_addr: Address, p_int: int, p_bytes: bytes, p_str: str):
        self.call_event_log(p_log_index, p_bool, p_addr, p_int, p_bytes, p_str)
        recipient_score = self.create_interface_score(_to, InterCallInterface)
        recipient_score.call_event_log(p_log_index, p_bool, p_addr, p_int, p_bytes, p_str)
