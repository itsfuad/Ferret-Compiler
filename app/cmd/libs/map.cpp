#include <cstdlib>
#include <cstring>

// implement map with only raw c 
class Map {
    private:
    struct Entry {
        const char* key;
        const char* value;
    };
    Entry* entries;
    public:
    size_t size;
    size_t capacity;

    // methods
    Map(size_t initial_capacity) {
        capacity = initial_capacity;
        size = 0;
        entries = (Entry*)malloc(capacity * sizeof(Entry));
    }

    ~Map() {
        free(entries);
    }

    void put(const char* key, const char* value) {
        if (size == capacity) {
            capacity *= 2;
            entries = (Entry*)realloc(entries, capacity * sizeof(Entry));
        }
        entries[size++] = {key, value};
    }

    const char* get(const char* key) const {
        for (size_t i = 0; i < size; i++) {
            if (strcmp(entries[i].key, key) == 0) {
                return entries[i].value;
            }
        }
        return nullptr;
    }
};