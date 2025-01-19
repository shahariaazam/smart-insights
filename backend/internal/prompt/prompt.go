package prompt

import (
	"fmt"
	"strings"
)

// LLMPayload contains the data needed for generating prompts
type LLMPayload struct {
	DBSchema        string
	Question        string
	InitialQuery    string
	QueryResultJSON string
}

// InitialPrompt generates the prompt for SQL query generation
func (l *LLMPayload) InitialPrompt() string {
	return fmt.Sprintf(`You are a PostgreSQL expert who helps convert natural language questions into SQL queries.
Your task is to analyze the provided database schema and generate the most appropriate SQL query to answer the user's question.

Database Schema:
"""
%s
"""

User Question: %s

Instructions:
1. Analyze the schema carefully, considering table relationships and available columns
2. Generate a single, efficient SQL query that answers the user's question without inline comment
3. Use appropriate JOINs when needed. Don't make up any join if not needed or the table doesn't exists
4. Include WHERE clauses to filter data appropriately
5. Use aggregations (GROUP BY, HAVING) when required for summary data
6. Order results in a logical way using ORDER BY when appropriate
7. Limit results if returning large datasets
8. Consider query performance and optimization

Important Notes:
- Ensure the query follows PostgreSQL syntax
- Use lowercase for SQL keywords for consistency
- Include proper table aliases when joining multiple tables
- Add appropriate comments for complex logic
- Handle NULL values appropriately
- Only include tables and columns that exist in the schema
- No DML operations (INSERT, UPDATE, DELETE) allowed

Response Format (no markdown):
<sql>
Your SQL query here
</sql>

Generate the SQL query now.`, l.DBSchema, l.Question)
}

// GenerateReportPrompt creates the prompt for formatting query results
func (l *LLMPayload) GenerateReportPrompt() string {
	return fmt.Sprintf(`You are a reporting assistant skilled in converting database query results into clear,
markdown-formatted reports. Based on the provided JSON data and the user's question, create an appropriate
markdown report that effectively visualizes and explains the data.

User Question: %s

Query Results (JSON):
%s

Instructions:
1. Analyze the data structure and values carefully
2. Choose the most appropriate format for presentation:
   - Tables for structured, columnar data
   - Lists for enumerated items
   - Summaries for aggregated data
   - Charts or graphs (using markdown syntax) when appropriate
3. Include relevant statistics or insights
4. Format numbers appropriately (e.g., currencies, percentages)
5. Keep the report concise but informative
6. Use proper markdown syntax and formatting insie the <markdown> tags. Don't use markdown response outside of <markdown> tags.'

Response Format (no markdown):
<markdown>
Your markdown-formatted report here
</markdown>

Generate the report now.`, l.Question, l.QueryResultJSON)
}

// ExtractResponse extracts content between specified XML-style tags
func ExtractResponse(tag, response string) string {
	startTag := fmt.Sprintf("<%s>", tag)
	endTag := fmt.Sprintf("</%s>", tag)

	startIndex := strings.Index(response, startTag)
	endIndex := strings.Index(response, endTag)

	if startIndex == -1 || endIndex == -1 {
		return ""
	}

	return strings.TrimSpace(response[startIndex+len(startTag) : endIndex])
}

// ValidateQuery performs basic validation of generated SQL query
func ValidateQuery(query string) error {
	query = strings.ToLower(strings.TrimSpace(query))

	// Check for dangerous operations
	dangerousKeywords := []string{"drop", "truncate", "delete", "update", "insert", "alter", "create"}
	for _, keyword := range dangerousKeywords {
		if strings.Contains(query, keyword) {
			return fmt.Errorf("query contains forbidden keyword: %s", keyword)
		}
	}

	// Ensure it starts with SELECT
	if !strings.HasPrefix(query, "select") {
		return fmt.Errorf("query must start with SELECT")
	}

	return nil
}
