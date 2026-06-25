import json
import os

input_file = 'data/longmemeval_oracle.json'
output_file = 'data/benchmarks/longmemeval/dataset.json'

with open(input_file, 'r', encoding='utf-8') as f:
    data = json.load(f)

print(f"Loaded {len(data)} items from {input_file}")

out_questions = []
out_memories = []
memory_seen = set()

# Limiting to 50 for a quick benchmark test unless specified otherwise.
for item in data[:50]:
    q_id = item['question_id']
    q_type = item.get('question_type', 'general')
    question = item['question']
    ground_truth = item.get('answer', '')
    
    expected_memory_id = ""
    
    # Process sessions
    for session_id_idx, session_messages in enumerate(item.get('haystack_sessions', [])):
        session_id = item['haystack_session_ids'][session_id_idx]
        
        has_answer_in_session = False
        # Combine messages into one text
        session_text_lines = []
        for msg in session_messages:
            session_text_lines.append(f"{msg['role']}: {msg['content']}")
            if msg.get('has_answer', False):
                has_answer_in_session = True
                expected_memory_id = session_id
                
        session_text = "\n".join(session_text_lines)
        
        mem_id = session_id
        if mem_id not in memory_seen:
            memory_seen.add(mem_id)
            out_memories.append({
                "id": mem_id,
                "content": session_text,
                "user_id": "benchmark-user"
            })
            
    out_questions.append({
        "id": q_id,
        "question": question,
        "session_id": "s001",
        "category": q_type,
        "ground_truth": ground_truth,
        "memory_id": expected_memory_id
    })

print(f"Generated {len(out_questions)} questions and {len(out_memories)} memories")

out_dataset = {
    "name": "longmemeval",
    "description": "Converted from longmemeval_oracle.json",
    "source": "local",
    "questions": out_questions,
    "memories": out_memories
}

# Ensure directory exists
os.makedirs(os.path.dirname(output_file), exist_ok=True)

with open(output_file, 'w', encoding='utf-8') as f:
    json.dump(out_dataset, f, indent=2)

print(f"Saved to {output_file}")
