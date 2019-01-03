# Copyright 2018 ICON Foundation
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

import importlib.util
import json
import sys
from os import path


class IconScoreLoader(object):
    _PACKAGE_JSON_FILE = 'package.json'
    _MAIN_SCORE = 'main_score'
    _MAIN_FILE = 'main_file'

    @staticmethod
    def _load_json(score_path: str) -> dict:
        pkg_json_path = path.join(score_path, IconScoreLoader._PACKAGE_JSON_FILE)
        with open(pkg_json_path, 'r') as f:
            return json.load(f)

    @staticmethod
    def load_package(score_path: str) -> callable:
        if not path.exists(score_path):
            return None
        score_package_info = IconScoreLoader._load_json(score_path)
        dirname: str = path.dirname(score_path)
        package: str = path.basename(score_path)
        if dirname not in sys.path:
            sys.path.append(dirname)

        # in order for the new module to be noticed by the import system
        importlib.invalidate_caches()
        module = importlib.import_module(f".{score_package_info[IconScoreLoader._MAIN_FILE]}", package)

        return getattr(module, score_package_info[IconScoreLoader._MAIN_SCORE])
