-- Seed 3 slash commands mẫu cho user đầu tiên (user_id = 1)
-- Chỉ insert nếu chưa tồn tại để migration idempotent

INSERT OR IGNORE INTO prompts (user_id, title, content, category, tags, is_slash, slash_name)
VALUES
  (1,
   'Email Writer',
   'Write a professional email about {{.input}}. Tone: {{.tone}}. Language: {{.lang}}.',
   'productivity',
   'email,writing',
   1,
   'email'),

  (1,
   'Code Review',
   'Review the following code and provide feedback on quality, bugs, and improvements:

{{.input}}

Focus on: readability, performance, security, and best practices.',
   'development',
   'code,review',
   1,
   'review'),

  (1,
   'Translate',
   'Translate the following text to {{.lang}}:

{{.input}}

Keep the original tone and formatting.',
   'language',
   'translate,language',
   1,
   'translate');
