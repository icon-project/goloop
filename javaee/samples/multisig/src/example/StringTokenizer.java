/*
 * Copyright 2020 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package example;

import java.util.NoSuchElementException;

/**
 * A simplified version of the java.util.StringTokenizer
 *
 * Neither support surrogates nor return delimiters as a token.
 */
public class StringTokenizer {
    private int currentPosition;
    private int newPosition;
    private final int maxPosition;
    private final String str;
    private final String delimiters;

    public StringTokenizer(String str, String delim) {
        currentPosition = 0;
        newPosition = -1;
        this.str = str;
        maxPosition = str.length();
        delimiters = delim;
    }

    public boolean hasMoreTokens() {
        newPosition = skipDelimiters(currentPosition);
        return (newPosition < maxPosition);
    }

    public String nextToken() {
        currentPosition = (newPosition >= 0) ?
                newPosition : skipDelimiters(currentPosition);
        newPosition = -1;

        if (currentPosition >= maxPosition)
            throw new NoSuchElementException();
        int start = currentPosition;
        currentPosition = scanToken(currentPosition);
        return str.substring(start, currentPosition);
    }

    private int skipDelimiters(int startPos) {
        if (delimiters == null) {
            throw new NullPointerException();
        }
        int position = startPos;
        while (position < maxPosition) {
            char c = str.charAt(position);
            if (delimiters.indexOf(c) < 0)
                break;
            position++;
        }
        return position;
    }

    private int scanToken(int startPos) {
        int position = startPos;
        while (position < maxPosition) {
            char c = str.charAt(position);
            if (delimiters.indexOf(c) >= 0)
                break;
            position++;
        }
        return position;
    }
}
