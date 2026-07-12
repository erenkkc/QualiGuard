/**
 * Kapadokya Taş Konak - Ana JavaScript Dosyası
 * Tüm etkileşimler client-side olarak çalışır.
 * Inline JavaScript kullanılmaz; addEventListener ile event bağlanır.
 */

document.addEventListener('DOMContentLoaded', function () {
  initStickyNav();
  initHamburgerMenu();
  initSmoothScroll();
  initReservationForm();
  initRoomFilter();
  initGalleryLightbox();
});

/**
 * Sticky menü scroll efekti
 * Sayfa belirli mesafe kaydırıldığında header'a gölge ve arka plan değişikliği uygular.
 */
function initStickyNav() {
  var header = document.querySelector('.site-header');
  if (!header) return;

  window.addEventListener('scroll', function () {
    if (window.scrollY > 50) {
      header.classList.add('scrolled');
    } else {
      header.classList.remove('scrolled');
    }
  });
}

/**
 * Hamburger menü toggle
 * Mobil cihazlarda navigasyon menüsünü açar/kapatır.
 */
function initHamburgerMenu() {
  var hamburger = document.querySelector('.hamburger');
  var nav = document.querySelector('.main-nav');
  if (!hamburger || !nav) return;

  hamburger.addEventListener('click', function () {
    hamburger.classList.toggle('active');
    nav.classList.toggle('open');
    var isOpen = nav.classList.contains('open');
    hamburger.setAttribute('aria-expanded', isOpen);
  });

  var navLinks = nav.querySelectorAll('a');
  navLinks.forEach(function (link) {
    link.addEventListener('click', function () {
      hamburger.classList.remove('active');
      nav.classList.remove('open');
      hamburger.setAttribute('aria-expanded', 'false');
    });
  });
}

/**
 * Smooth scroll
 * Sayfa içi anchor bağlantılarında akıcı kaydırma sağlar.
 */
function initSmoothScroll() {
  var anchors = document.querySelectorAll('a[href^="#"]');
  anchors.forEach(function (anchor) {
    anchor.addEventListener('click', function (e) {
      var targetId = this.getAttribute('href');
      if (targetId === '#') return;

      var target = document.querySelector(targetId);
      if (target) {
        e.preventDefault();
        target.scrollIntoView({ behavior: 'smooth', block: 'start' });
      }
    });
  });
}

/**
 * Dinamik bildirim gösterir
 * @param {string} message - Gösterilecek mesaj metni
 * @param {string} type - Bildirim tipi: 'success' veya 'error'
 */
function showNotification(message, type) {
  var existing = document.querySelector('.notification');
  if (existing) existing.remove();

  var notification = document.createElement('div');
  notification.className = 'notification ' + type;
  notification.setAttribute('role', 'alert');
  notification.textContent = message;
  document.body.appendChild(notification);

  requestAnimationFrame(function () {
    notification.classList.add('show');
  });

  setTimeout(function () {
    notification.classList.remove('show');
    setTimeout(function () {
      notification.remove();
    }, 400);
  }, 4000);
}

/**
 * Rezervasyon formu doğrulama ve gönderim
 * Tüm alanları kontrol eder, tarih geçerliliğini doğrular ve başarı bildirimi gösterir.
 */
function initReservationForm() {
  var form = document.getElementById('reservation-form');
  if (!form) return;

  form.addEventListener('submit', function (e) {
    e.preventDefault();
    if (validateReservationForm(form)) {
      showNotification('Rezervasyonunuz başarıyla alındı! En kısa sürede sizinle iletişime geçeceğiz.', 'success');
      form.reset();
      clearFormErrors(form);
    }
  });

  var checkIn = form.querySelector('#check-in');
  var checkOut = form.querySelector('#check-out');

  if (checkIn && checkOut) {
    checkIn.addEventListener('change', function () {
      validateDates(checkIn, checkOut);
    });
    checkOut.addEventListener('change', function () {
      validateDates(checkIn, checkOut);
    });
  }
}

/**
 * Form alanlarını doğrular
 * @param {HTMLFormElement} form - Doğrulanacak form elementi
 * @returns {boolean} Form geçerliyse true döner
 */
function validateReservationForm(form) {
  var isValid = true;
  clearFormErrors(form);

  var requiredFields = form.querySelectorAll('[required]');
  requiredFields.forEach(function (field) {
    if (!field.value.trim()) {
      markFieldError(field, 'Bu alan zorunludur.');
      isValid = false;
    }
  });

  var email = form.querySelector('#email');
  if (email && email.value.trim()) {
    var emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailPattern.test(email.value.trim())) {
      markFieldError(email, 'Geçerli bir e-posta adresi giriniz.');
      isValid = false;
    }
  }

  var phone = form.querySelector('#phone');
  if (phone && phone.value.trim()) {
    var phonePattern = /^[\d\s\-\+\(\)]{10,}$/;
    if (!phonePattern.test(phone.value.trim())) {
      markFieldError(phone, 'Geçerli bir telefon numarası giriniz.');
      isValid = false;
    }
  }

  var checkIn = form.querySelector('#check-in');
  var checkOut = form.querySelector('#check-out');
  if (checkIn && checkOut && checkIn.value && checkOut.value) {
    if (!validateDates(checkIn, checkOut)) {
      isValid = false;
    }
  }

  if (!isValid) {
    showNotification('Lütfen formdaki hataları düzeltiniz.', 'error');
  }

  return isValid;
}

/**
 * Giriş ve çıkış tarihlerini karşılaştırır
 * Giriş tarihi çıkış tarihinden önce olmalıdır.
 * @param {HTMLInputElement} checkIn - Giriş tarihi inputu
 * @param {HTMLInputElement} checkOut - Çıkış tarihi inputu
 * @returns {boolean} Tarihler geçerliyse true döner
 */
function validateDates(checkIn, checkOut) {
  var checkInDate = new Date(checkIn.value);
  var checkOutDate = new Date(checkOut.value);
  var today = new Date();
  today.setHours(0, 0, 0, 0);

  if (checkIn.value && checkInDate < today) {
    markFieldError(checkIn, 'Giriş tarihi bugünden önce olamaz.');
    showNotification('Giriş tarihi bugünden önce olamaz.', 'error');
    return false;
  }

  if (checkIn.value && checkOut.value && checkInDate >= checkOutDate) {
    markFieldError(checkOut, 'Çıkış tarihi giriş tarihinden sonra olmalıdır.');
    showNotification('Çıkış tarihi, giriş tarihinden sonra olmalıdır.', 'error');
    return false;
  }

  clearFieldError(checkIn);
  clearFieldError(checkOut);
  return true;
}

/**
 * Hatalı alanı işaretler ve hata mesajı gösterir
 * @param {HTMLElement} field - Hatalı form alanı
 * @param {string} message - Hata mesajı
 */
function markFieldError(field, message) {
  field.classList.add('error');
  field.setAttribute('aria-invalid', 'true');

  var errorEl = field.parentElement.querySelector('.error-message');
  if (errorEl) {
    errorEl.textContent = message;
    errorEl.classList.add('visible');
  }
}

/**
 * Alan hata işaretini temizler
 * @param {HTMLElement} field - Temizlenecek form alanı
 */
function clearFieldError(field) {
  field.classList.remove('error');
  field.removeAttribute('aria-invalid');

  var errorEl = field.parentElement.querySelector('.error-message');
  if (errorEl) {
    errorEl.textContent = '';
    errorEl.classList.remove('visible');
  }
}

/**
 * Formdaki tüm hata işaretlerini temizler
 * @param {HTMLFormElement} form - Temizlenecek form
 */
function clearFormErrors(form) {
  var fields = form.querySelectorAll('.error');
  fields.forEach(function (field) {
    clearFieldError(field);
  });
}

/**
 * Oda filtreleme
 * Kullanıcının girdiği fiyat ve kapasite bilgisine göre oda kartlarını filtreler.
 */
function initRoomFilter() {
  var filterForm = document.getElementById('room-filter');
  if (!filterForm) return;

  var maxPriceInput = document.getElementById('max-price');
  var capacitySelect = document.getElementById('capacity-filter');
  var roomCards = document.querySelectorAll('.room-card');

  function applyFilter() {
    var maxPrice = maxPriceInput ? parseInt(maxPriceInput.value, 10) : Infinity;
    var capacity = capacitySelect ? capacitySelect.value : 'all';

    if (isNaN(maxPrice)) maxPrice = Infinity;

    var visibleCount = 0;
    roomCards.forEach(function (card) {
      var cardPrice = parseInt(card.dataset.price, 10);
      var cardCapacity = card.dataset.capacity;

      var priceMatch = cardPrice <= maxPrice;
      var capacityMatch = capacity === 'all' || cardCapacity === capacity;

      if (priceMatch && capacityMatch) {
        card.classList.remove('hidden');
        visibleCount++;
      } else {
        card.classList.add('hidden');
      }
    });

    if (visibleCount === 0) {
      showNotification('Seçtiğiniz kriterlere uygun oda bulunamadı.', 'error');
    }
  }

  if (maxPriceInput) {
    maxPriceInput.addEventListener('input', applyFilter);
  }
  if (capacitySelect) {
    capacitySelect.addEventListener('change', applyFilter);
  }

  var resetBtn = document.getElementById('filter-reset');
  if (resetBtn) {
    resetBtn.addEventListener('click', function () {
      if (maxPriceInput) maxPriceInput.value = '';
      if (capacitySelect) capacitySelect.value = 'all';
      roomCards.forEach(function (card) {
        card.classList.remove('hidden');
      });
    });
  }
}

/**
 * Galeri lightbox
 * Galeri görsellerine tıklandığında modal pencerede büyütülmüş görüntü gösterir.
 */
function initGalleryLightbox() {
  var galleryItems = document.querySelectorAll('.gallery-item');
  if (galleryItems.length === 0) return;

  var lightbox = document.createElement('div');
  lightbox.className = 'lightbox';
  lightbox.setAttribute('role', 'dialog');
  lightbox.setAttribute('aria-label', 'Görsel büyütme');
  lightbox.innerHTML =
    '<button class="lightbox-close" aria-label="Kapat">&times;</button>' +
    '<img src="" alt="">' +
    '<p class="lightbox-caption"></p>';
  document.body.appendChild(lightbox);

  var lightboxImg = lightbox.querySelector('img');
  var lightboxCaption = lightbox.querySelector('.lightbox-caption');
  var closeBtn = lightbox.querySelector('.lightbox-close');

  galleryItems.forEach(function (item) {
    item.addEventListener('click', function () {
      var img = item.querySelector('img');
      if (!img) return;

      lightboxImg.src = img.src;
      lightboxImg.alt = img.alt;
      lightboxCaption.textContent = img.alt;
      lightbox.classList.add('active');
      document.body.style.overflow = 'hidden';
      closeBtn.focus();
    });

    item.addEventListener('keydown', function (e) {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        item.click();
      }
    });
  });

  function closeLightbox() {
    lightbox.classList.remove('active');
    document.body.style.overflow = '';
    lightboxImg.src = '';
  }

  closeBtn.addEventListener('click', closeLightbox);

  lightbox.addEventListener('click', function (e) {
    if (e.target === lightbox) closeLightbox();
  });

  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape' && lightbox.classList.contains('active')) {
      closeLightbox();
    }
  });
}
