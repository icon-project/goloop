# Copyright 2019 ICON Foundation
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

import logging
import os
from enum import Flag
from logging import Logger as builtinLogger, FileHandler, Formatter
from typing import Union


class OutputType(Flag):
    NONE = 0
    CONSOLE = 1
    FILE = 2
    CUSTOM = 4


class LogConfig:
    default_fmt = "[%(levelname)s|%(filename)s:%(lineno)s] %(asctime)s > %(message)s"

    def __init__(self,
                 name: str,
                 level: str,
                 file_path: str,
                 fmt: str,
                 output_type: 'OutputType'):

        self.name: str = name
        self.level: str = level
        self.file_path: str = file_path
        self.fmt: str = fmt
        self.output_type: 'OutputType' = output_type

    @classmethod
    def from_dict(cls, src_config: dict):
        config: dict = src_config.get('log')
        if config is None:
            return

        name: str = config.get('name', "Logger")
        level: str = config.get('level', 'info').upper()
        file_path: str = config.get('filePath', "")
        fmt: str = config.get('format', cls.default_fmt)
        output_type: 'OutputType' = OutputType.NONE

        output_types: str = config.get('outputType')
        if output_types:
            outputs = output_types.split('|')
            for output in outputs:
                output_type |= OutputType[output.upper()]

        return LogConfig(name, level, file_path, fmt, output_type)


class LoggerUtil(object):
    _formatter: 'Formatter' = None

    @classmethod
    def apply_config(cls, logger: 'builtinLogger', config: dict, handler=None):
        log_config: 'LogConfig' = LogConfig.from_dict(config)

        logger.handlers.clear()
        logger.name = log_config.name
        logger.setLevel(log_config.level)
        cls._apply_config(logger, log_config, handler)

    @classmethod
    def print_config(cls, logger: 'builtinLogger', config: dict):
        logger.info(f'====================LOG CONFIG START====================')
        cls._view_config_info(logger, config, "CONFIG")
        logger.info(f'====================LOG CONFIG END======================')

    @classmethod
    def _view_config_info(cls, logger: 'builtinLogger', conf: dict, prefix: str):
        for key, value in conf.items():
            if not isinstance(value, dict):
                tmp_prefix = '{}.{}'.format(prefix, key)
                logger.info(f'[{tmp_prefix}] > {value}')
            else:
                tmp_prefix = '{}.{}'.format(prefix, key)
                cls._view_config_info(logger, value, tmp_prefix)

    @classmethod
    def _apply_config(cls, logger: 'builtinLogger', log_config: 'LogConfig', custom_handler=None):
        cls._formatter = Formatter(log_config.fmt)

        if cls._is_flag_on(log_config.output_type, OutputType.CONSOLE):
            handler = logging.StreamHandler()
            handler.setFormatter(cls._formatter)
            logger.addHandler(handler)

        if cls._is_flag_on(log_config.output_type, OutputType.FILE):
            cls._ensure_dir(log_config.file_path)
            handler = FileHandler(log_config.file_path, 'a')
            handler.setFormatter(cls._formatter)
            logger.addHandler(handler)

        if cls._is_flag_on(log_config.output_type, OutputType.CUSTOM):
            if custom_handler:
                handler = custom_handler
                handler.setFormatter(cls._formatter)
                logger.addHandler(handler)

    @classmethod
    def _is_flag_on(cls, src_flag: 'Flag', dest_flag: 'Flag') -> bool:
        return src_flag & dest_flag == dest_flag

    @classmethod
    def _ensure_dir(cls, file_path: str):
        directory = os.path.dirname(file_path)
        if not os.path.exists(directory):
            os.makedirs(directory)

    @classmethod
    def make_log_msg(cls, tag: str, msg: Union[str, BaseException]):
        return f'[{tag}] {msg}'
