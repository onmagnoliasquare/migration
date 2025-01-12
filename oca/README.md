# OCA Migration

**Goal**: The goal of this migration is to transfer On Century Avenue (OCA) posts, author, tag, category, and multimedia data over to Sanity. The transformation process follows these steps:

1. Create an SQL database locally to query for data
2. Create queries that fetch relevant information
3. Store output from the queries in structs
4. Translate the structs into `json` format, which conforms to our new Sanity schema
5. Format the schema into an `ndjson` file
6. Upload this file to Sanity using Sanity's CLI tool

The order of data upload follows this sequence:

1. Authors
2. Tags and Categories
3. Media
4. Posts

Posts may have all the previous three types of data, and therefore require these fields to be populated to have a complete and valid article document in Sanity Content Lake.

## Tools and Software

Tools and software used to accomplish this migration:

- **Docker/Compose** to run a local MySQL database.
- **DataGrip** for easy access to browsing SQL data.
- **Go** for the main transformation program.
- **Node** for running JavaScript files.
- **Transnomino** for batch renaming of files

### Disambiguation

Lots of posts have revision or autosave as their name. The latest revision should be used to transfer the data instead of any other previous revision. The latest revision can be found using the post date.

There are some posts that have not been posted.

Posts have images attached to them. To know if an attachment is just a single post and not just an image post, check the `post_parent` column. This lets you link many rows together.

Original posts have a `post_parent` of `0`.

Extract the slug from GUID

## Process

Wordpress stores its data in SQL format using MySQL. This means the data can be queried and ascertained in a sane, programmatic way. To use MySQL, a `compose.yaml` file is used to setup a docker container to host the MySQL server locally. This server can then be queried using DataGrip or any other program that can execute a query.

## Finding relevant data

The first order of business is to sort this data into their respective categories.

One way Wordpress models data is with a "parent" and a "child" relationship between two rows existing in the same table. Parents are a column in a table, usually in the form of `column_name_parent`. The column is of the type `integer`, and therefore is a number referring to the `ID` field of the parent. There is no `column_name_child` column because being a child is an implicit property; if a row has a parent, they therefore are a child. So, a parent row can be a child of another parent row and have their own children rows as well.

So where does the lineage terminate? A row with a parent field with the value of `0` is the top-level parent, which, throughout this document, will be referred to as the **root** parent.

### Posts

Because Wordpress has revisions and autosaves, all of which are stored in the **posts** table, it is necessary to filter those out and find what is actually published and posted. To find those, query for rows that have a post `post_parent` of 0, a `post_type` of "post", and a `post_status` of "publish". The SQL query is below. Remember to change `FROM`'s argument to your actual table name. This is just the table name for our database.

```sql
-- get published posts
SELECT *
FROM wp_x7zvdw3xj9_posts
WHERE post_parent = 0 AND post_type = 'post' AND post_status = 'publish'
ORDER BY post_date;
```

To find post attachments, like images, a similar query can be used:

```sql
-- get post media attachments
SELECT *
FROM wp_x7zvdw3xj9_posts
WHERE post_parent = 0 AND post_type = 'attachment'
ORDER BY post_date;
```

Notice that the `post_status` field is not filtered, because the `post_status` for [attachments are naturally "inherit"](https://wordpress.org/documentation/article/post-status/#inherit).

To link the filenames to the corresponding data in the table, the column GUID can be used. It contains a URL which can be truncated to just the resource path, which is conveniently sorted by year and month. If the media export from wordpress has been exported in a sorted format, which exports by year and month, linking the two together is as simple as going to that year and month directory and finding the appropriate file name.

### Authors

The next piece of data needed is author data (any individuals who have contributed to the corpus on the previous website), which can be found in two places, which are the **users** table and the **usermeta** table. The **usermeta** table can be queried by `user_id` using the `ID` column found in the **users** table to get individual user metadata.

There's a lot of inconsistencies with the data. Some users do not have first or last names filled in. Some have nicknames that are either their Net ID or their actual name. Some have a login that is either their real name or their Net ID. Another issue is the jerry-rigged workaround of making a new user composed of two names. This happened because they were unable to link an article to multiple authors.

### Tags and Categories of a post

Wordpress uses a blanket term for tags, categories, navbar snippets, anything of such ilk: _terms_ is what they are called. Both tags and categories, ways the end user can find content, are both classified as terms. The relevant tables for finding term related data are the **term_relationships**, **term_taxonomy**, and **terms** tables. Let's find the category of a particular post.

#### Step 1

A reasonable place to start is to find the category names by which the site progenitors decided to categorize posts. To find terms that are categories, query the **\_term_taxonomy** table:

```sql
SELECT *
FROM wp_x7zvdw3xj9_term_taxonomy
WHERE taxonomy = 'category';
```

Alternatively, the query can be adjusted to query for only the root parent categories. Simply add the `parent` parameter.

```sql
WHERE taxonomy = 'category' AND parent = 0;
```

#### Step 2

The output retrieves two columns of interest: `term_taxonomy_id` and `term_id`. To find the names of the categories, values from the `term_id` column will be used to build a query for the **\_term** table. The query is below. Remember that the values will differ based on the `term_id`s in your database.

```sql
SELECT *
FROM wp_x7zvdw3xj9_terms
WHERE term_id IN (1,17,84,85,86,91,92,93,94,95,122,130,525,570,716,939);
```

The numbers are a comma separated list of the `term_id`s, no spaces between each number. The relevant rows returned have the `name` column, which is the category name.

#### Step 3

Now, to find posts under a certain category, query the **\_term_relationships**. This table contains a column titled `object_id`, which is corresponds to the column `ID` in the **\_posts** table. Below is the query:

```sql
SELECT *
FROM wp_x7zvdw3xj9_term_relationships
WHERE term_id = 84;
```

For our database, `84` is the `News` category. Replace with an arbitrary `term_id` value returned in **Step 1**.

The number of rows returned varies based on the dataset. If you have any, write down an arbitrary row's `object_id` column.

#### Step 4

Use this value to query the `ID` column in the **\_posts** table. This should output the post row that the `object_id` corresponds to. You now know the category of a post in your database.

The aforementioned sequence of steps applies to any **term** in the **\_taxonomy** table. To find a **term** with the value post_tag, replace the taxonomy value in the SQL query `WHERE` clause with `'post_tag'`.

### All associated tags and the category of a post

This is simpler than the above process. Query the **\_term_relationships** table with a post's `ID`:

```sql
SELECT *
FROM wp_x7zvdw3xj9_term_relationships
WHERE object_id = [post ID]
```

`[post id]` is the `ID` of interest. The output should return none or many values in the `term_taxonomy_id` column. Repeat **Step 2** with the values returned in the query to find the names of the categories or tags.

## Media sanitization and upload

The Wordpress media is sorted into directories by year, then month. A media file may have a path like:

`2018/03/campus_map.jpg`

Because media has to be uploaded first before posts, the filenames must be sanitized/reformatted into a more informative and workable naming scheme. This is the final filename format:

`2018-03-campus_map.jpg`

Quite simple, but the complications happen when trying to connect the Wordpress content's `<img>` `src` attributes to these files. A typical `src` value would be something like:

`http://oncenturyavenue.org/wp-content/uploads/2018/02/media-20180214-3-1024x847.png`

The only thing in common is the filepath, which is the year, then month. The filename is similar, however, the image's dimensions are infixed between the original name and the dot before the file extension.

## Sanity upload commands

```bash
yarn sanity dataset import oca.ndjson [dataset]
```

## Troubleshooting

### `drafts.(id)`

Make sure that when importing a Sanity dataset, **there are no draft ids**.

## Useful Links

### Wordpress

- [Wordpress Taxonomy Terms](https://taxopress.com/taxonomy-terms-data-wordpress-database/)
- [Taxonomy Diagram](https://codex.wordpress.org/images/7/75/Z2Ohv.png)

### Docker

- [Docker MySQL](https://hub.docker.com/_/mysql)
- [Login to MySQL on Docker and run an SQL file](https://dev.to/n350071/login-to-mysql-on-docker-and-run-a-sql-file-2bk7)
- [Init script on Docker compose](https://iamvickyav.medium.com/mysql-init-script-on-docker-compose-e53677102e48)

### HTML Parsing

- [Migrate HTML content from Ghost - Sanity Blog](https://dev.to/n350071/login-to-mysql-on-docker-and-run-a-sql-file-2bk7)
- [Sanity Block Tools - NPM Package](https://www.npmjs.com/package/@sanity/block-tools?activeTab=readme)
