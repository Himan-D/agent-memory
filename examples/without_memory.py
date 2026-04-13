# Without Agent Memory - Basic AI Agent

from openai import OpenAI

client = OpenAI(api_key="your-api-key")

# Simple conversation - NO MEMORY
messages = [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "My name is John and I work at Acme Corp."},
    {
        "role": "assistant",
        "content": "Nice to meet you, John! How can I help you today?",
    },
    {"role": "user", "content": "What company do I work for?"},
]

response = client.chat.completions.create(model="gpt-4", messages=messages)

print(response.choices[0].message.content)
# Output: "I don't have access to information about your employment unless you tell me."

# Every conversation starts FRESH - no context of previous interactions
# The AI cannot remember:
# - User's name
# - Previous questions asked
# - Facts about the user
# - Preferences or history
