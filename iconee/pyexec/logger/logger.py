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
import sys

from logging import DEBUG, INFO, WARNING, ERROR, currentframe
from .utils import LoggerUtil

default_logger = logging.Logger("python")

# This code is mainly copied from the python logging module, with minor modifications

# _srcfile is used when walking the stack to check when we've got the first
# caller stack frame.
#

if hasattr(sys, 'frozen'): #support for py2exe
    _srcfile = "logging%s__init__%s" % (os.sep, __file__[-4:])
elif __file__[-4:].lower() in ['.pyc', '.pyo']:
    _srcfile = __file__[:-4] + '.py'
else:
    _srcfile = __file__
_srcfile = os.path.normcase(_srcfile)


class Logger(object):
    @classmethod
    def load_config(cls, config: dict, handler=None):
        LoggerUtil.apply_config(default_logger, config, handler)

    @classmethod
    def print_config(cls, config: dict):
        LoggerUtil.print_config(default_logger, config)

    @classmethod
    def isDebugEnabled(cls) -> bool:
        return default_logger.isEnabledFor(DEBUG)

    @classmethod
    def debug(cls, msg: str, tag: str = "DEBUG"):
        if default_logger.isEnabledFor(DEBUG):
            cls._log(DEBUG, LoggerUtil.make_log_msg(tag, msg))

    @classmethod
    def info(cls, msg: str, tag: str = "INFO"):
        if default_logger.isEnabledFor(INFO):
            cls._log(INFO, LoggerUtil.make_log_msg(tag, msg))

    @classmethod
    def warning(cls, msg: str, tag: str = "WARN"):
        if default_logger.isEnabledFor(WARNING):
            # redirect warning to info
            cls.info(msg, tag)

    @classmethod
    def error(cls, msg: str, tag: str = "ERROR"):
        if default_logger.isEnabledFor(ERROR):
            # redirect error to info
            cls.info(msg, tag)

    @classmethod
    def exception(cls, msg: str, tag: str = "LOG"):
        if default_logger.isEnabledFor(DEBUG):
            cls._log(INFO, LoggerUtil.make_log_msg(tag, msg), exc_info=True)
        elif default_logger.isEnabledFor(INFO):
            cls._log(INFO, LoggerUtil.make_log_msg(tag, msg))

    @classmethod
    def _log(cls, level, msg, args=None, exc_info=None, extra=None):
        """
        Low-level logging routine which creates a LogRecord and then calls
        all the handlers of this logger to handle the record.
        """
        # Add wrapping functionality here.
        if _srcfile:
            # IronPython doesn't track Python frames, so findCaller throws an
            # exception on some versions of IronPython. We trap it here so that
            # IronPython can use logging.
            try:
                fn, lno, func = cls.findCaller()
            except ValueError:
                fn, lno, func = "(unknown file)", 0, "(unknown function)"
        else:
            fn, lno, func = "(unknown file)", 0, "(unknown function)"
        if exc_info:
            if not isinstance(exc_info, tuple):
                exc_info = sys.exc_info()
        record = default_logger.makeRecord(
            default_logger.name, level, fn, lno, msg, args, exc_info, func, extra)
        default_logger.handle(record)

    @classmethod
    def findCaller(cls):
        """
        Find the stack frame of the caller so that we can note the source
        file name, line number and function name.
        """
        f = currentframe()
        # On some versions of IronPython, currentframe() returns None if
        # IronPython isn't run with -X:Frames.
        if f is not None:
            f = f.f_back
        rv = "(unknown file)", 0, "(unknown function)"
        while hasattr(f, "f_code"):
            co = f.f_code
            filename = os.path.normcase(co.co_filename)
            if filename == _srcfile:
                f = f.f_back
                continue
            rv = (co.co_filename, f.f_lineno, co.co_name)
            break
        return rv


class SystemLogger(Logger):
    @classmethod
    def exception(cls, msg: str, tag: str = "LOG"):
        if default_logger.isEnabledFor(WARNING):
            cls._log(WARNING, LoggerUtil.make_log_msg(tag, msg), exc_info=True)
