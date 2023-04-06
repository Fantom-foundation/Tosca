#include "common/word.h"

#include <array>
#include <cstdint>
#include <ostream>

namespace tosca {

static_assert(std::is_trivial_v<Word>);
static_assert(std::is_trivially_assignable_v<Word, Word>);
static_assert(sizeof(Word) == 32);

Word::Word(std::initializer_list<const std::uint8_t> list) {
  for (std::size_t i = 0; i < list.size() && i < data_.size(); i++) {
    data_[i] = static_cast<std::byte>(std::data(list)[i]);
  }
  for (std::size_t i = list.size(); i < data_.size(); i++) {
    data_[i] = std::byte(0);
  }
}

std::ostream& operator<<(std::ostream& out, const Word& word) {
  constexpr auto toSymbol = [](char x) -> char { return x < 10 ? '0' + x : 'A' + x - 10; };
  for (auto cur : word.data_) {
    out << toSymbol(static_cast<char>(cur >> 4));
    out << toSymbol(static_cast<char>(cur) & 0xF);
  }
  return out;
}

}  // namespace tosca
