#!/usr/bin/env python
import os
import re
import sys
import getopt
import math
import struct

sys.path.append(os.path.dirname(sys.argv[0]))
import sqlid

__date__ = 't'.join('$Date: 2011/09/26 20:04:12 $'.split(' ')[1:3])
__doc__= """
Usage: %s [OPTIONS] [script...]

Calculate the SQL ID of the string(s) provided.

Note: This SQL ID uses the same algorithm as the DB SQL ID but makes
      no attempt to produce the same value as the DB SQL ID.
      The purpose of this SQL ID is to accurately identify the same
      query regardless of with-clause aliases and constant literals.

  -a, --hash         Only provide the sql hash.
  -i, --id           Only provide the sql id.
  -N, --no-name      Don't output the name of the file.
  -I, --case         Case sensitive.
  -C, --no-uncomnent Don't uncomment the string prior to
                     calculation. Comments are C /**/ style.
  -Z, --no-compress  Don't compress the string prior to calculation.
  -L, --no-newline   Strip trailing carriage returns.
  -W, --keep-with    Keep with-clauses intact
  -R, --keep-const   Keep string and number literals
  -S, --semicolon    Keep trailing semi-colon
  -h, --help         This help.
  -t, --tsv, --tabs  Use tabs in output instead of spaces.
  -F, --format=STR   Format string.
                       i = sql id
                       h = sql hash
                       n = input name
                       q = compressed string
                       s = original string

  build: %s""" % (os.path.basename(sys.argv[0]),__date__)

def usage():
  global __doc__
  print __doc__

# Trailing newline
trailingre = re.compile("\n[ \t\v\f]*$")

def main():
  try:
    opts, args = getopt.getopt(sys.argv[1:],
                      "aCF:iIhLNo:RStvWxZ",
                      ["help",
                        "format=",                # -F
                        "as-tsv", "tabs", "tsv",  # -t
                        "case",                   # -I
                        "no-compress",            # -ZC
                        "no-uncomment",           # -C
                        "no-newline",             # -L
                        "keep-wtih",              # -W
                        "keep-const",             # -R
                        "semicolon",
                        "semi-colon"              # -S
                        "only-hash", "hash",      # -a
                        "only-id", "id", "sqlid", # -i
                        "no-name",                # -N
                        "no-stdin=",              # -x
                        "output=",                # -o
                        "verbose"])
  except getopt.GetoptError, err:
    # print help information and exit:
    print str(err) # will print something like "option -a not recognized"
    usage()
    sys.exit(2)
  output = None
  formatted = False
  format = "i\th\tn"
  case = False
  sep = " "
  verbose = False
  compress = True
  uncomment = True
  semicolon = False
  newline = True
  no_name = False
  keep_with = False
  keep_const = False
  only_hash = False
  only_id = False
  stdin = True
  for o, a in opts:
    if o in ("-v", "--verbose"):
      verbose = True
    elif o in ("-h", "--help"):
      usage()
      sys.exit()
    elif o in ("-F", "--format"):
      formatted = True
      format = a
    elif o in ("-t", "--as-tsv", "--tabs", "--tsv"):
      sep = "\t"
    elif o in ("-I", "--case"):
      case = True
    elif o in ("-Z", "--no-compress"):
      compress = False
      uncomment = False
    elif o in ("-C", "--no-uncomment"):
      uncomment = False
    elif o in ("-S", "--semicolon", "--semi-colon"):
      semicolon = True
    elif o in ("-L", "--no-newline"):
      newline = False
    elif o in ("-W", "--keep-with"):
      keep_with = True
    elif o in ("-R", "--keep-const"):
      keep_const = True
    elif o in ("-N", "--no-name"):
      no_name = True
    elif o in ("-a", "--hash", "--only-hash"):
      only_hash = True
    elif o in ("-i", "--sqlid", "--id", "--only-id"):
      only_id = True
    elif o in ("-x", "--no-stdin"):
      stdin = False
    elif o in ("-o", "--output"):
      output = a
    else:
      assert False, "unhandled option"

  stdin=stdin and not sys.stdin.isatty()
  if stdin:
    args.append(sys.stdin.read())

  l=len(args)
  for i in range(0,l):

    arg = args[i]
    if not arg.strip():
      continue

    name = ''
    if os.path.exists(arg):
      name = arg
      sql = file(arg).read()
    else:
      sql = arg
      if i==(l-1) and stdin:
        name = '--'
      else:
        name = "arg[%d]" % i

    if no_name:
      name=''

    osql = sql
    sql = sqlid.compress(sql,
                         do_compress=compress,
                         uncomment=uncomment,
                         nosemicolon=not semicolon,
                         newline=newline,
                         nocase=not case,
                         rewith=not keep_with,
                         noconst=not keep_const)


    sid = sqlid.sqlid_raw(sql)
    shs = sqlid.sqlhash_raw(sql)
      
    if formatted:
      format = format.replace("\\n","\n")
      format = format.replace("\\t","\t")
      for f in format:
        if 'i' == f:
          sys.stdout.write(sid)
        elif 'h' == f:
          sys.stdout.write("%s" % shs)
        elif 'n' == f:
          sys.stdout.write(name)
        elif f in ('q','c'):
          sys.stdout.write(sqlid.compress(sql))
        elif 's' == f:
          sys.stdout.write(osql)
        else:
          sys.stdout.write("%c" % f)
    elif 1 == l and stdin:
      if only_id:
        print sid
      elif only_hash:
        print shs
      else:
        print "%s%s%s" % (sid, sep, shs)
    elif only_id:
      print "%s%s%s" % (sid, sep, name)
    elif only_hash:
      print "%s%s%s" % (shs, sep, name)
    elif not verbose:
      print "%s%s%s%s%s" % (sid, sep, shs, sep, name)
    else:
      print "%s%s%s%s%s%s%s" % (sid, sep, shs, sep, name, sep, sql)


if __name__ == "__main__":
    main()

sys.exit(0)
