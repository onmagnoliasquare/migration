"use-strict";

import { htmlToBlocks, normalizeBlock } from "@sanity/block-tools";
import { Schema } from "@sanity/schema";
import { JSDOM } from "jsdom";
import path from "path";
import fs from "fs";

const processHtmlFile = (filePath, outputFilePath) => {
  try {
    // Read the HTML file
    const fullPath = path.resolve(filePath);
    const htmlContent = fs.readFileSync(fullPath, "utf8");

    // Replace \r\n with <br> and \" with "
    const modifiedContent = htmlContent
      .replace(/\u00A0/g, " ")
      .replace(/\\r\\n/g, "<br>")
      .replace(/\\n/g, "<br>")
      .replace(/\\"/g, '"')
      .replace(/&nbsp;/g, "<br>");

    // Save the modified content to the output file
    fs.writeFileSync(outputFilePath, modifiedContent, "utf8");
    // console.log(`Processed HTML saved to ${outputFilePath}`);
  } catch (error) {
    console.error(`Error processing file at ${filePath}:`, error);
  }
};

// Input and output file paths
// These file paths are from the perspective of main.go.
const inputHtmlPath = "./js/index.html";
const outputHtmlPath = "./js/output/transformed_block_content.html";
const transformedBlockContent = "./js/output/transformed_block_output.json";

// const inputHtmlPath = process.argv[0];
// const outputHtmlPath = process.argv[1];
// const transformedBlockContent = process.argv[2];

// Process the file
processHtmlFile(inputHtmlPath, outputHtmlPath);

let htmlContent;

const filePath = path.resolve(outputHtmlPath);
try {
  htmlContent = fs.readFileSync(filePath, "utf8");
} catch (error) {
  console.error(`Error reading file at ${filePath}:`, error);
  process.exit(1);
}

/**
 * The schema here is from the website repository, under the directory
 * `packages/backend/schemaTypes/objects/blockContent.tsx`
 */
const defaultSchema = Schema.compile({
  name: "backend",
  types: [
    {
      type: "object",
      name: "content",
      fields: [
        {
          title: "Title",
          type: "string",
          name: "title",
        },
        {
          title: "Body",
          name: "body",
          type: "array",
          of: [
            {
              type: "block",
              styles: [
                { title: "Normal", value: "normal" },
                { title: "Heading 1", value: "h2" },
                { title: "Heading 2", value: "h3" },
                { title: "Heading 3", value: "h4" },
                { title: "Quote", value: "blockquote" },
                { title: "Hidden", value: "blockComment" },
              ],
              marks: {
                decorators: [
                  { title: "Strong", value: "strong" },
                  { title: "Emphasis", value: "em" },
                  { title: "Superscript", value: "superscript" },
                  { title: "Subscript", value: "subscript" },
                  { title: "Underline", value: "underline" },
                ],
              },
            },
          ],
        },
      ],
    },
    {
      type: "image",
      fields: [
        {
          name: "title",
          title: "Title",
          type: "string",
        },
        {
          name: "alt",
          title: "Alt Text",
          type: "string",
        },
      ],
    },
  ],
});

// The compiled schema type for the content type that holds the block array
const blockContentType = defaultSchema
  .get("content")
  .fields.find((field) => field.name === "body").type;

// const htmlContent = getHtmlArgument();

// Convert HTML to block array
// Replace all \r\n with <br>
// Then replace all \" with "
// Replace \r\n with <br> and \" with "
// Only allow 1 <br> between elements. This removes extra <br>'s.
const blocks = htmlToBlocks(htmlContent, blockContentType, {
  parseHtml: (html) => new JSDOM(html).window.document,
  rules: [
    {
      // Makes <br> turn into ""
      deserialize(el, next, block) {
        if (el.nodeName.toLowerCase() !== "br") {
          return undefined;
        }
        return normalizeBlock(
          block({
            style: "normal",
            markDefs: [],
            children: [
              {
                _type: "span",
                marks: [],
                text: "",
              },
            ],
            _type: "block",
          })
        );
      },
    },
    {
      // <img>
      deserialize(el, next, block) {
        if (el.nodeName.toLowerCase() !== "img") {
          return undefined;
        }

        // If the <img> element has an src that is a `googleusercontent.com`
        // related domain, return a text block rather than an image one. Some of
        // the links here are dead and must be changed manually. Turning it into
        // a text block makes it obvious which links must be converted.

        if (el.getAttribute("src").includes("googleusercontent")) {
          return normalizeBlock(
            block({
              style: "normal",
              markDefs: [],
              children: [
                {
                  _type: "span",
                  marks: [],
                  text: el.getAttribute("src"),
                },
              ],
              _type: "block",
            })
          );
        }

        // Otherwise, return a normal image type block.

        return normalizeBlock(
          block({
            _type: "image",
            alt: el.getAttribute("alt") || "replace this alt text",
            asset: {
              // This will be resolved to the real reference in the Go code.
              _ref: el.getAttribute("src") || "",
              _type: "reference",
            },
          })
        );
      },
    },
  ],
});

const writeBlocksToFile = (filePath, data) => {
  const fullPath = path.resolve(filePath);

  try {
    // Serialize data as JSON and write to file
    fs.writeFileSync(fullPath, JSON.stringify(data, null, 2), "utf8");
    console.log(JSON.stringify(data, null, 2));
    // console.log(`Blocks written to ${fullPath}`);
  } catch (error) {
    console.error(`Error writing to file at ${fullPath}:`, error);
  }
};

// Write the blocks data to the JSON file
writeBlocksToFile(transformedBlockContent, blocks);
