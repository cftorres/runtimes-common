#!/usr/bin/python

# Copyright 2017 Google Inc. All rights reserved.

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

# http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import logging
import unittest
import urlparse

import test_util


class TestRoot(unittest.TestCase):

    def __init__(self, url, methodName='runTest'):
        self._url = urlparse.urljoin(url, test_util.ROOT_ENDPOINT)
        super(TestRoot, self).__init__()

    def runTest(self):
        logging.debug('Hitting endpoint: {0}'.format(self._url))
        output, status_code = test_util.get(self._url)
        logging.info('output is: {0}'.format(output))
        self.assertEquals(status_code, 0,
                          'Cannot connect to sample application!')

        self.assertEquals(output, test_util.ROOT_EXPECTED_OUTPUT,
                          'Unexpected output: expected {0}, received {1}'
                          .format(test_util.ROOT_EXPECTED_OUTPUT, output))
