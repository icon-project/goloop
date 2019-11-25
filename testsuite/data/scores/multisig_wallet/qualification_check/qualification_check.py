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

from iconservice import *


def only_wallet(func):
    if not isfunction(func):
        revert(f"{func} is not a function.")

    @wraps(func)
    def __wrapper(calling_obj: object, *args, **kwargs):
        if calling_obj.msg.sender != calling_obj.address:
            revert(f"{func} method can be called only by the wallet SCORE (address: {calling_obj.address})")

        return func(calling_obj, *args, **kwargs)
    return __wrapper
