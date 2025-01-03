#!/bin/bash
# Turn all the JSON files into NDJSON format, and create a master file with all
# records.

cat ../output/transformed_oca_users.json | jq -c '.[]' > ../output/transformed_oca_users.ndjson
cat ../output/transformed_oca_tags.json | jq -c '.[]' >> ../output/transformed_oca_tags.ndjson
cat ../output/transformed_oca_articles.json | jq -c '.[]' >> ../output/transformed_oca_articles.ndjson



cat ../output/transformed_oca_users.json | jq -c '.[]' > ../output/oca.ndjson
cat ../output/transformed_oca_tags.json | jq -c '.[]' >> ../output/oca.ndjson
cat ../output/transformed_oca_articles.json | jq -c '.[]' >> ../output/oca.ndjson
