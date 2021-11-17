# Copyright 2021 ICON Foundation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import unittest

from pyexec.iconscore.icon_score_step import StepType, IconScoreStepCounter


class TestStepCounter(unittest.TestCase):

    def setUp(self):
        self.step_costs_v0 = {
            'get': 0,
            'set': 320,
            'replace': 80,
            'delete': -240,
            'eventLog': 100,
            'apiCall': 10000
        }
        self.step_costs_v1 = {
            'schema': 1,
            'get': 80,
            'getBase': 2000,
            'set': 320,
            'setBase': 5000,
            'delete': -240,
            'deleteBase': 3000,
            'log': 200,
            'logBase': 5000,
            'apiCall': 10000
        }
        self.step_limit = 1_000_000_000

    def _get_step_counter(self, version: int = 0):
        _step_costs = self.step_costs_v1 if version == 1 else self.step_costs_v0
        return IconScoreStepCounter(_step_costs, self.step_limit,
                                    self._dummy_refund_handler)

    def _dummy_refund_handler(self):
        pass

    def test_step_costs_v0(self):
        step_costs = self._get_step_counter()._step_costs
        for key, value in step_costs.items():
            self.assertEqual(value, self.step_costs_v0[key.value])

    def test_step_costs_v1(self):
        step_costs = self._get_step_counter(1)._step_costs
        for key, value in step_costs.items():
            self.assertEqual(value, self.step_costs_v1[key.value])


if __name__ == '__main__':
    unittest.main()
