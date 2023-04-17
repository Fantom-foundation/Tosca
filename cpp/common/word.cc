#include "common/word.h"

#include <bit>
#include <ranges>

#include "intx/intx.hpp"

namespace tosca {

static_assert(std::is_trivial_v<Word>);
static_assert(std::is_trivially_assignable_v<Word, Word>);
static_assert(std::endian::native == std::endian::little, "Only supporting little-endian systems so far");
static_assert(sizeof(Word) == 32);

static_assert(sizeof(intx::uint256) == sizeof(Word));

Word::Word(std::initializer_list<const std::uint8_t> list) {
  auto dst = data_.begin();
  auto src = std::rbegin(list);
  for (; dst != data_.end() && src != std::rend(list); ++dst, ++src) {
    *dst = static_cast<std::byte>(*src);
  }
  std::fill(dst, data_.end(), std::byte{0});
}

std::byte Word::operator[](std::uint8_t offset) const {
  if (offset < data_.size()) {
    return data_[(data_.size() - 1u) - offset];
  } else {
    return std::byte{0};
  }
}

std::byte Word::operator[](const Word& offset) const {
  const auto size = static_cast<std::uint8_t>(data_.size());

  if (offset < Word{size}) {
    return data_[(size - 1u) - static_cast<std::uint8_t>(offset.data_[0])];
  } else {
    return std::byte{0};
  }
}

std::strong_ordering Word::operator<=>(const Word& other) const {
  for (std::size_t i = data_.size() - 1; i > 0; --i) {
    if (data_[i] != other.data_[i]) {
      return data_[i] <=> other.data_[i];
    }
  }
  return data_[0] <=> other.data_[0];
}

Word Word::operator+(const Word& other) const {
  auto a = std::bit_cast<intx::uint256>(*this);
  auto b = std::bit_cast<intx::uint256>(other);
  return std::bit_cast<Word>(a + b);
}

Word Word::operator-(const Word& other) const {
  auto a = std::bit_cast<intx::uint256>(*this);
  auto b = std::bit_cast<intx::uint256>(other);
  return std::bit_cast<Word>(a - b);
}

Word Word::operator*(const Word& other) const {
  auto a = std::bit_cast<intx::uint256>(*this);
  auto b = std::bit_cast<intx::uint256>(other);
  return std::bit_cast<Word>(a * b);
}

Word Word::operator/(const Word& denom) const {
  if (denom == Word{0}) {
    return Word{0};
  }

  auto a = std::bit_cast<intx::uint256>(*this);
  auto b = std::bit_cast<intx::uint256>(denom);
  return std::bit_cast<Word>(a / b);
}

Word Word::operator%(const Word& denom) const {
  if (denom == Word{0}) {
    return Word{0};
  }

  auto a = std::bit_cast<intx::uint256>(*this);
  auto b = std::bit_cast<intx::uint256>(denom);
  return std::bit_cast<Word>(a % b);
}

Word Word::operator<<(const Word& shift) const {
  auto a = std::bit_cast<intx::uint256>(*this);
  auto b = std::bit_cast<intx::uint256>(shift);
  return std::bit_cast<Word>(a << b);
}

Word Word::operator>>(const Word& shift) const {
  auto a = std::bit_cast<intx::uint256>(*this);
  auto b = std::bit_cast<intx::uint256>(shift);
  return std::bit_cast<Word>(a >> b);
}

Word Word::operator|(const Word& other) const {
  auto a = std::bit_cast<intx::uint256>(*this);
  auto b = std::bit_cast<intx::uint256>(other);
  return std::bit_cast<Word>(a | b);
}

Word Word::operator&(const Word& other) const {
  auto a = std::bit_cast<intx::uint256>(*this);
  auto b = std::bit_cast<intx::uint256>(other);
  return std::bit_cast<Word>(a & b);
}

Word Word::operator^(const Word& other) const {
  auto a = std::bit_cast<intx::uint256>(*this);
  auto b = std::bit_cast<intx::uint256>(other);
  return std::bit_cast<Word>(a ^ b);
}

Word Word::operator~() const {
  auto a = std::bit_cast<intx::uint256>(*this);
  return std::bit_cast<Word>(~a);
}

Word Word::SignedDiv(const Word& denom) const {
  if (denom == Word{0}) {
    return Word{0};
  }

  auto a = std::bit_cast<intx::uint256>(*this);
  auto b = std::bit_cast<intx::uint256>(denom);
  return std::bit_cast<Word>(intx::sdivrem(a, b).quot);
}

Word Word::SignedMod(const Word& denom) const {
  if (denom == Word{0}) {
    return Word{0};
  }

  auto a = std::bit_cast<intx::uint256>(*this);
  auto b = std::bit_cast<intx::uint256>(denom);
  return std::bit_cast<Word>(intx::sdivrem(a, b).rem);
}

Word Word::Exp(const Word& exponent) const {
  auto a = std::bit_cast<intx::uint256>(*this);
  auto b = std::bit_cast<intx::uint256>(exponent);
  return std::bit_cast<Word>(intx::exp(a, b));
}

std::ostream& operator<<(std::ostream& out, const Word& word) {
  constexpr auto toSymbol = [](char x) -> char { return x < 10 ? '0' + x : 'A' + x - 10; };
  for (auto cur : word.data_ | std::views::reverse) {
    out << toSymbol(static_cast<char>(cur >> 4));
    out << toSymbol(static_cast<char>(cur) & 0xF);
  }
  return out;
}

}  // namespace tosca
