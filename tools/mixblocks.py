#!/usr/bin/env python2
# -*- coding: latin-1 -*-

"""
Mix schedule blocks
"""

import sys

def error(msg, code=1):
    print "Error: %s" % msg
    sys.exit(code)

def main():
    if len(sys.argv) != 2:
        print "Usage: %s blocks.txt" % sys.argv[0]
        sys.exit(2)

    f = open(sys.argv[1])
    data = f.readlines()
    f.close()

    k = -1 
    blocks = []
    for line in data:
        line = line[:-1].strip()
        if line == '' or line.startswith('#'):
            continue
        if line.startswith('h='):
            blocks.append(list())
            line = line[2:]
            k += 1
        if k == -1:
            error("missing h= definition")
        blocks[k].append(line)

    if len(blocks) > 1:
        nblines = len(blocks[0])
        out = []
        for j in range(len(blocks)):
            if len(blocks[j]) != nblines:
                error("wrong line count for block %d: got %d, expect %d" % (j+1, len(blocks[j]), nblines))

        for j in range(nblines):
            out.append(list())
            for k in range(len(blocks)):
                out[j].append(blocks[k][j])

        tmp = [' '.join(line) for line in out]
        print '\n'.join(tmp)
    else:
        print '\n'.join(blocks[0])

if __name__ == '__main__':
    main()

