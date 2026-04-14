SELECT subject, template
FROM {{schema}}.system_email_template
WHERE context = $1 AND iso_639_1 = $2