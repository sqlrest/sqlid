#!/usr/bin/env python
# Usage:
#
import os
import re
import sys
import getopt
import math
import struct

try:
  import hashlib
  class md:
    @staticmethod
    def digest(stmt):
      return hashlib.md5(stmt + '\x00').digest()

except ImportError:
  import md5
  class md:
    @staticmethod
    def digest(stmt):
      return md5.new(stmt + '\x00').digest()

# Match C-style comments that aren't like SQL hints: /*+ ... */
uncommentre = re.compile("/[*][^+](?:(?![*]/).)*[*]/")
# Whitespace
compressre = re.compile("[ \t\n\r\v\f]+")
# Trailing semi-colon
semicolonre = re.compile(";[ \t\n\r\v\f]*$")
# with-clauses
withre = re.compile("^(?:with\s+|,\s*)([^\s.]+)\s+as",re.IGNORECASE)
# String literals, includes support for embedded double-single-quotes
#stringre = re.compile("'((?:[^']+|'')*)'(?!')") # Horrible performance! Avoid!
stringre = re.compile("'((?:[^']+|'')*)('?)(?!')")
#stringre = re.compile("'[^']+'")
#multiqre = re.compile("[?]{2,}")
numberre = re.compile("\s\d+\s")
# Quote
quotere = re.compile("^q'(.)")

def id2hash(sqlid):
  sum = 0
  i = 1
  alphabet = '0123456789abcdfghjkmnpqrstuvwxyz'
  for ch in sqlid:
    sum += alphabet.index(ch) * (32**(len(sqlid) - i))
    i += 1
  return sum % (2 ** 32)

class Node(list):
  def __init__(self, iterable=(), **attributes):
    list.__init__(self, iterable)

  def __repr__(self):
    return '(%s)' % ''.join([str(x) for x in self])

nstack=[]

def nesting(stmt):
  global nstack
  
  root = Node()
  nstack.append(root)

  skip = -1
  start = end = 0

  l = len(stmt)
  
  while end < l:
    c = stmt[end]
    end += 1

    # Skip quoted strings.
    # Don't be concerned with double-single-quotes. It'll skip those too.
    if '\'' == c or '"' == c:
      if -1 == skip:
        skip = c
      elif c == skip:
        skip = -1

    # TODO: handle quote string: q'{ 'any' text }'

    if -1 != skip:
      continue

    if '(' == c or ')' == c:
      top = nstack[-1]
      top.append(stmt[start:end-1])

      #print "%c %3d %3d head %30s tail %30s %30s" % (c,start,end,'[%s]' % stmt[start:end-1],stmt[end:-1],top)

      if '(' == c:
        n = Node()
        nstack.append(n)
      else:
        top = nstack.pop()
        nstack[-1].append(top)

      start = end

  if start != end:
    root.append(stmt[start:-1])

  return root

def compress(stmt, do_compress=True, uncomment=True, nosemicolon=True, newline=True, nocase=True, rewith=True, noconst=True):
  if nocase:
    stmt = stmt.lower()
  if uncomment:
    stmt = uncommentre.sub("",stmt)
  if nosemicolon:
    stmt = semicolonre.sub("",stmt)
  if do_compress:
    stmt = compressre.sub(" ",stmt).strip()
  if newline:
    stmt = "".join([stmt,"\n"])
  if rewith and withre.search(stmt):
    seq=1
    for n in nesting(stmt):
      if not isinstance(n,list):
        for m in withre.finditer(n):
          stmt = stmt.replace("%s " % m.group(1),"^%04X^ " % seq)
          seq += 1
  if noconst:
    stmt = stringre.sub("?",stmt)
    #stmt = multiqre.sub("?",stmt) # Was used before stringre was fixed to perform better.
    stmt = numberre.sub(" ? ",stmt)
  return stmt

def sqlid(stmt, do_compress=True, uncomment=True, nosemicolon=True, newline=True, nocase=True, rewith=True, noconst=True):
  if compress:
    stmt = compress(stmt, 
                    do_compress=do_compress,
                    uncomment=uncomment,
                    nosemicolon=nosemicolon,
                    newline=newline,
                    nocase=nocase,
                    rewith=rewith,
                    noconst=noconst)
  return sqlid_raw(stmt)

def sqlid_raw(stmt):
  #sys.stdout.write('[%s]' % stmt)
  h = md.digest(stmt)
  (d1,d2,msb,lsb) = struct.unpack('IIII', h)
  sqln = msb * (2 ** 32) + lsb
  stop = int(math.log(sqln, math.e) / math.log(32, math.e) + 1)
  sqlid = ''
  alphabet = '0123456789abcdfghjkmnpqrstuvwxyz'
  for i in range(0, stop):
    sqlid = alphabet[(sqln / (32 ** i)) % 32] + sqlid
  return sqlid

def sqlhash(stmt, do_compress=True, uncomment=True, nosemicolon=True, newline=True, nocase=True, rewith=True, noconst=True):
  if compress:
    stmt = compress(stmt,
                    do_compress=do_compress,
                    uncomment=uncomment,
                    nosemicolon=nosemicolon,
                    newline=newline,
                    nocase=nocase,
                    rewith=rewith,
                    noconst=noconst)
  return sqlhash_raw(stmt)

def sqlhash_raw(stmt):
  return struct.unpack('IIII', md.digest(stmt))[3]
