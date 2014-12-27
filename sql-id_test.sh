#!/bin/bash
go test -v
python sql-id.py -RL -F 'i s\n' 'select 1' 'select * from table'
