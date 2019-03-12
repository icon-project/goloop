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

import importlib
import json
import os
import sys

from ..base.exception import IllegalFormatException

PACKAGE_JSON_FILE = 'package.json'
MAIN_MODULE = 'main_module'
MAIN_SCORE = 'main_score'


class IconScoreLoader(object):

    @staticmethod
    def _load_package_json(score_path: str) -> dict:
        pkg_json_path = os.path.join(score_path, PACKAGE_JSON_FILE)
        with open(pkg_json_path, 'r') as f:
            return json.load(f)

    @staticmethod
    def _get_package_info(package_json: dict) -> tuple:
        main_module: str = package_json.get(MAIN_MODULE)
        if not isinstance(main_module, str):
            # "main_file" field will be deprecated soon.
            # Use "main_module" instead
            main_module: str = package_json['main_file']

        # Relative package name is not allowed
        if main_module.startswith('.'):
            raise IllegalFormatException('Invalid main_module')

        main_score: str = package_json[MAIN_SCORE]

        return main_module, main_score

    @staticmethod
    def load_module(score_path: str) -> callable:
        if not os.path.exists(score_path):
            return None
        dirname: str = os.path.dirname(score_path)
        package: str = os.path.basename(score_path)
        if dirname not in sys.path:
            sys.path.append(dirname)

        package_json = IconScoreLoader._load_package_json(score_path)
        main_module, main_score = IconScoreLoader._get_package_info(package_json)

        # in order for the new module to be noticed by the import system
        importlib.invalidate_caches()
        module = importlib.import_module(f".{main_module}", package)

        return getattr(module, main_score)
