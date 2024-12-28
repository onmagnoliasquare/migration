FROM mysql:9.0
LABEL authors="Neo"


# For some reason, this isn't working, so I (Neo)
# used DataGrip to import the data into the running
# MySQL database.
COPY ../scripts/1.sql /docker-entrypoint-initdb.d/
